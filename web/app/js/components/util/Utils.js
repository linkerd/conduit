import _ from 'lodash';
import React from 'react';
import * as d3 from 'd3';

/*
* Display grid constants
*/
export const baseWidth = 8; // design base width of 8px
export const rowGutter = 3 * baseWidth;

/*
* Number formatters
*/
const successRateFormatter = d3.format(".2%");
const latencyFormatter = d3.format(",");

export const metricToFormatter = {
  "REQUEST_RATE": m => _.isNil(m) ? "---" : styleNum(m, " RPS", true),
  "SUCCESS_RATE": m => _.isNil(m) ? "---" : successRateFormatter(m),
  "LATENCY": m => `${_.isNil(m) ? "---" : latencyFormatter(m)} ms`
};

/*
* Add commas to a number (converting it to a string in the process)
*/
export function addCommas(nStr) {
  nStr += '';
  let x = nStr.split('.');
  let x1 = x[0];
  let x2 = x.length > 1 ? '.' + x[1] : '';
  let rgx = /(\d+)(\d{3})/;
  while (rgx.test(x1)) {
    x1 = x1.replace(rgx, '$1' + ',' + '$2');
  }
  return x1 + x2;
}

/*
* Round a number to a given number of decimals
*/
export const roundNumber = (num, dec) => {
  return Math.round(num * Math.pow(10,dec)) / Math.pow(10,dec);
};

/*
* Shorten and style number
*/
export const styleNum = (number, unit = "", truncate = true) => {
  if (Number.isNaN(number)) {
    return "N/A";
  }

  if (truncate && number > 999999999) {
    number = roundNumber(number / 1000000000.0, 1);
    return addCommas(number) + "G" + unit;
  } else if (truncate && number > 999999) {
    number = roundNumber(number / 1000000.0, 1);
    return addCommas(number) + "M" + unit;
  } else if (truncate && number > 999) {
    number = roundNumber(number / 1000.0, 1);
    return addCommas(number) + "k" + unit;
  } else if (number > 999) {
    number = roundNumber(number, 0);
    return addCommas(number) + unit;
  } else {
    number = roundNumber(number, 2);
    return addCommas(number) + unit;
  }
};

/*
* Convert a string to a valid css class name
*/
export const toClassName = name => {
  if (!name) return "";
  return _.lowerCase(name).replace(/[^a-zA-Z0-9]/g, "_");
};

/*
* Instructions for adding deployments to service mesh
*/
export const instructions = name => {
  if (name) {
    return (
      <div className="action">Add {name} to the deployment.yml file<br /><br />
      Then run <code>conduit inject deployment.yml | kubectl apply -f -</code> to add it to the service mesh</div>
    );
  } else {
    return (
      <div className="action">Add one or more deployments to the deployment.yml file<br /><br />
      Then run <code>conduit inject deployment.yml | kubectl apply -f -</code> to add them to the service mesh</div>
    );
  }
};
