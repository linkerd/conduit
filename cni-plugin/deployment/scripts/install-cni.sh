#!/bin/sh
# Copyright (c) 2018 Tigera, Inc. All rights reserved.
# Copyright 2018 Istio Authors
# Modifications copyright (c) Linkerd authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This file was inspired by:
# 1) https://github.com/projectcalico/cni-plugin/blob/c1175467c227c1656577c80bfc0ee7795da2e2bc/k8s-install/scripts/install-cni.sh
# 2) https://github.com/istio/cni/blob/c63a509539b5ed165a6617548c31b686f13c2133/deployments/kubernetes/install/scripts/install-cni.sh

# Script to install Linkerd CNI on a Kubernetes host.
# - Expects the host CNI binary path to be mounted at /host/opt/cni/bin.
# - Expects the host CNI network config path to be mounted at /host/etc/cni/net.d.
# - Expects the desired CNI config in the CNI_NETWORK_CONFIG env variable.

# Ensure all variables are defined, and that the script fails when an error is hit.
set -u -e

# Helper function for raising errors
# Usage:
# some_command || exit_with_error "some_command_failed: maybe try..."
exit_with_error() {
  echo "${1}"
  exit 1
}

# The mount point of the host machine. Defaults to /host, but can be
# overridden by setting MOUNT_POINT
HOST_MOUNT_POINT=${MOUNT_POINT:-/host}

# The directory on the host where CNI plugin configs are installed. Defaults to
# /etc/cni/net.d, but can be overridden by setting CNI_NET_DIR.
HOST_CNI_NET_DIR=${CNI_NET_DIR:-/etc/cni/net.d}
# The directory on the host where the CNI binaries are installed. Defaults to
# /opt/cni/bin, but can be overridden by setting CNI_BIN_DIR
HOST_CNI_BIN_DIR=${CNI_BIN_DIR:-/opt/cni/bin}

# Default to the first file following a find | sort since the Kubernetes CNI runtime is going
# to look for the lexicographically first file. If the directory is empty, then use a name
# of our choosing.
CNI_CONF_PATH=${CNI_CONF_PATH:-$(find "${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}" -maxdepth 1 -type f \( -iname '*conflist' -o -iname '*conf' \) | sort | head -n 1)}
CNI_CONF_PATH=${CNI_CONF_PATH:-"${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}/01-linkerd-cni.conflist"}

KUBECONFIG_FILE_NAME=${KUBECONFIG_FILE_NAME:-ZZZ-linkerd-cni-kubeconfig}

cleanup() {
  echo 'Removing linkerd-cni artifacts.'

  if [ -e "${CNI_CONF_PATH}" ]; then
    echo "Removing linkerd-cni config: ${CNI_CONF_PATH}"
    CNI_CONF_DATA=$(cat "${CNI_CONF_PATH}" | jq 'del( .plugins[]? | select( .type == "linkerd-cni" ))')
    echo "${CNI_CONF_DATA}" > "${CNI_CONF_PATH}"

    if [ "${CNI_CONF_PATH}" = "${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}/01-linkerd-cni.conflist" ]; then
      rm -f "${CNI_CONF_PATH}"
    fi
  fi
  if [ -e "${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}/${KUBECONFIG_FILE_NAME}" ]; then
    echo "Removing linkerd-cni kubeconfig: ${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}/${KUBECONFIG_FILE_NAME}"
    rm -f "${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}/${KUBECONFIG_FILE_NAME}"
  fi
  if [ -e "${HOST_MOUNT_POINT}${HOST_CNI_BIN_DIR}"/linkerd-cni ]; then
    echo "Removing linkerd-cni binary: ${HOST_MOUNT_POINT}${HOST_CNI_BIN_DIR}/linkerd-cni"
    rm -f "${HOST_MOUNT_POINT}${HOST_CNI_BIN_DIR}/linkerd-cni"
  fi
  echo 'Exiting.'
}

# Capture the usual signals and exit from the script
trap cleanup EXIT
trap 'echo "SIGINT received, simply exiting..."; cleanup' INT
trap 'echo "SIGTERM received, simply exiting..."; cleanup' TERM
trap 'echo "SIGHUP received, simply exiting..."; cleanup' HUP

# Place the new binaries if the directory is writeable.
dir="${HOST_MOUNT_POINT}${HOST_CNI_BIN_DIR}"
if [ ! -w "${dir}" ]; then
  echo "${dir} is non-writeable, skipping"
fi
for path in "${HOST_CNI_BIN_DIR}"/*;
do
  cp "${path}" "${dir}"/ || exit_with_error "Failed to copy ${path} to ${dir}."
done

echo "Wrote linkerd CNI binaries to ${dir}"

TMP_CONF='/linkerd-cni.conf.default'
# If specified, overwrite the network configuration file.
: "${CNI_NETWORK_CONFIG_FILE:=}"
: "${CNI_NETWORK_CONFIG:=}"
if [ -e "${CNI_NETWORK_CONFIG_FILE}" ]; then
  echo "Using CNI config template from ${CNI_NETWORK_CONFIG_FILE}."
  cp "${CNI_NETWORK_CONFIG_FILE}" "${TMP_CONF}"
elif [ "${CNI_NETWORK_CONFIG}" != "" ]; then
  echo 'Using CNI config template from CNI_NETWORK_CONFIG environment variable.'
  cat >"${TMP_CONF}" <<EOF
${CNI_NETWORK_CONFIG}
EOF
fi

SERVICE_ACCOUNT_PATH=/var/run/secrets/kubernetes.io/serviceaccount
KUBE_CA_FILE=${KUBE_CA_FILE:-${SERVICE_ACCOUNT_PATH}/ca.crt}
SKIP_TLS_VERIFY=${SKIP_TLS_VERIFY:-false}
# Pull out service account token.
SERVICEACCOUNT_TOKEN=$(cat ${SERVICE_ACCOUNT_PATH}/token)

# Check if we're running as a k8s pod.
if [ -f "${SERVICE_ACCOUNT_PATH}/token" ]; then
  # We're running as a k8d pod - expect some variables.
  if [ -z "${KUBERNETES_SERVICE_HOST}" ]; then
    echo 'KUBERNETES_SERVICE_HOST not set'; exit 1;
  fi
  if [ -z "${KUBERNETES_SERVICE_PORT}" ]; then
    echo 'KUBERNETES_SERVICE_PORT not set'; exit 1;
  fi

  if [ "${SKIP_TLS_VERIFY}" = "true" ]; then
    TLS_CFG='insecure-skip-tls-verify: true'
  elif [ -f "${KUBE_CA_FILE}" ]; then
    TLS_CFG="certificate-authority-data: $(cat "${KUBE_CA_FILE}" | base64 | tr -d '\n')"
  fi

  # Write a kubeconfig file for the CNI plugin. Do this
  # to skip TLS verification for now. We should eventually support
  # writing more complete kubeconfig files. This is only used
  # if the provided CNI network config references it.
  touch "${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}/${KUBECONFIG_FILE_NAME}"
  chmod "${KUBECONFIG_MODE:-600}" "${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}/${KUBECONFIG_FILE_NAME}"
  cat > "${HOST_MOUNT_POINT}${HOST_CNI_NET_DIR}/${KUBECONFIG_FILE_NAME}" <<EOF
# Kubeconfig file for linkerd CNI plugin.
apiVersion: v1
kind: Config
clusters:
- name: local
  cluster:
    server: ${KUBERNETES_SERVICE_PROTOCOL:-https}://[${KUBERNETES_SERVICE_HOST}]:${KUBERNETES_SERVICE_PORT}
    ${TLS_CFG}
users:
- name: linkerd-cni
  user:
    token: ${SERVICEACCOUNT_TOKEN}
contexts:
- name: linkerd-cni-context
  context:
    cluster: local
    user: linkerd-cni
current-context: linkerd-cni-context
EOF

fi

# Insert any of the supported "auto" parameters.
grep "__KUBERNETES_SERVICE_HOST__" ${TMP_CONF} && sed -i s/__KUBERNETES_SERVICE_HOST__/"${KUBERNETES_SERVICE_HOST}"/g ${TMP_CONF}
grep "__KUBERNETES_SERVICE_PORT__" ${TMP_CONF} && sed -i s/__KUBERNETES_SERVICE_PORT__/"${KUBERNETES_SERVICE_PORT}"/g ${TMP_CONF}
sed -i s/__KUBERNETES_NODE_NAME__/"${KUBERNETES_NODE_NAME:-$(hostname)}"/g ${TMP_CONF}
sed -i s/__KUBECONFIG_FILENAME__/"${KUBECONFIG_FILE_NAME}"/g ${TMP_CONF}
sed -i s/__CNI_MTU__/"${CNI_MTU:-1500}"/g ${TMP_CONF}

# Use alternative command character "~", since these include a "/".
sed -i s~__KUBECONFIG_FILEPATH__~"${HOST_CNI_NET_DIR}/${KUBECONFIG_FILE_NAME}"~g ${TMP_CONF}
sed -i s~__LOG_LEVEL__~"${LOG_LEVEL:-warn}"~g ${TMP_CONF}
sed -i s~__INCOMING_PROXY_PORT__~"${INCOMING_PROXY_PORT:=-1}"~g ${TMP_CONF}
sed -i s~__OUTGOING_PROXY_PORT__~"${OUTGOING_PROXY_PORT:=-1}"~g ${TMP_CONF}
sed -i s~__PROXY_UID__~"${PROXY_UID:=-1}"~g ${TMP_CONF}
sed -i s~__PORTS_TO_REDIRECT__~"${PORTS_TO_REDIRECT:=}"~g ${TMP_CONF}
sed -i s~__INBOUND_PORTS_TO_IGNORE__~"${INBOUND_PORTS_TO_IGNORE:=}"~g ${TMP_CONF}
sed -i s~__OUTBOUND_PORTS_TO_IGNORE__~"${OUTBOUND_PORTS_TO_IGNORE:=}"~g ${TMP_CONF}
sed -i s~__SIMULATE__~"${SIMULATE:=false}"~g ${TMP_CONF}

CNI_OLD_CONF_PATH="${CNI_OLD_CONF_PATH:-${CNI_CONF_PATH}}"

# Log the config file before inserting service account token.
# This way auth token is not visible in the logs.
echo "CNI config: $(cat ${TMP_CONF})"

sed -i s/__SERVICEACCOUNT_TOKEN__/"${SERVICEACCOUNT_TOKEN:-}"/g ${TMP_CONF}

CNI_CONF_FILE="${CNI_CONF_PATH}"
if [ -e "${CNI_CONF_FILE}" ]; then
  # Add the linkerd-cni plugin to the existing list
  CNI_TMP_CONF_DATA=$(cat "${TMP_CONF}")
  CNI_CONF_DATA=$(cat "${CNI_CONF_FILE}" | jq --argjson CNI_TMP_CONF_DATA "$CNI_TMP_CONF_DATA" -f /linkerd/filter.jq)
  echo "${CNI_CONF_DATA}" > ${TMP_CONF}
fi

# If the old config filename ends with .conf, rename it to .conflist, because it has changed to be a list
filename=$(basename -- "${CNI_CONF_PATH}")
extension="${filename##*.}"
if [ "${extension}" = ".conf" ]; then
  echo "Renaming ${CNI_CONF_PATH} extension to .conflist"
  CNI_CONF_PATH="${CNI_CONF_PATH}list"
fi

# Delete old CNI config files for upgrades.
if [ "${CNI_CONF_PATH}" != "${CNI_OLD_CONF_PATH}" ]; then
  echo "Removing CNI_OLD_CONF_PATH: ${CNI_OLD_CONF_PATH}"
  rm -f "${CNI_OLD_CONF_PATH}"
fi

# Move the temporary CNI config into place.
mv "${TMP_CONF}" "${CNI_CONF_PATH}" || exit_with_error 'Failed to mv files.'

echo "Created CNI config ${CNI_CONF_PATH}"

# Unless told otherwise, sleep forever.
# This prevents Kubernetes from restarting the pod repeatedly.
should_sleep=${SLEEP:-"true"}
echo "Done configuring CNI. Sleep=$should_sleep"
while [ "${should_sleep}" = "true"  ]; do
  sleep 10
done
