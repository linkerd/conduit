/* eslint-disable */
import 'raf/polyfill'; // the polyfill import must be first
import Adapter from 'enzyme-adapter-react-16';
import { ApiHelpers } from '../js/components/util/ApiHelpers.jsx';
import Enzyme from 'enzyme';
import { expect } from 'chai';
import { mount } from 'enzyme';
import { routerWrap } from './testHelpers.jsx';
import sinon from 'sinon';
import sinonStubPromise from 'sinon-stub-promise';
/* eslint-enable */

Enzyme.configure({ adapter: new Adapter() });
sinonStubPromise(sinon);

describe('ApiHelpers', () => {
  let api, fetchStub;

  beforeEach(() => {
    fetchStub = sinon.stub(window, 'fetch');
    fetchStub.returnsPromise().resolves({
      ok: true,
      json: () => Promise.resolve({ metrics: [] })
    });
    api = ApiHelpers("");
  });

  afterEach(() => {
    api = null;
    window.fetch.restore();
  });

  describe('getMetricsWindow/setMetricsWindow', () => {
    it('sets a default metricsWindow', () => {
      expect(api.getMetricsWindow()).to.equal('10m');
    });

    it('changes the metricsWindow on valid window input', () => {
      expect(api.getMetricsWindow()).to.equal('10m');

      api.setMetricsWindow('10s');
      expect(api.getMetricsWindow()).to.equal('10s');

      api.setMetricsWindow('1m');
      expect(api.getMetricsWindow()).to.equal('1m');

      api.setMetricsWindow('10m');
      expect(api.getMetricsWindow()).to.equal('10m');
    });

    it('does not change metricsWindow on invalid window size', () => {
      expect(api.getMetricsWindow()).to.equal('10m');

      api.setMetricsWindow('10h');
      expect(api.getMetricsWindow()).to.equal('10m');
    });
  });

  describe('ConduitLink', () => {
    it('wraps a relative link with the pathPrefix', () => {
      api = ApiHelpers('/my/path/prefix');
      let linkProps = { to: "/myrelpath", children: ["Informative Link Title"] };
      let conduitLink = mount(routerWrap(api.ConduitLink, linkProps));

      expect(conduitLink.find("Link")).to.have.length(1);
      expect(conduitLink.html()).to.contain('href="/my/path/prefix/myrelpath"');
      expect(conduitLink.html()).to.contain(linkProps.children[0]);
    });

    it('wraps a relative link with no pathPrefix', () => {
      api = ApiHelpers('');
      let linkProps = { to: "/myrelpath", children: ["Informative Link Title"] };
      let conduitLink = mount(routerWrap(api.ConduitLink, linkProps));

      expect(conduitLink.find("Link")).to.have.length(1);
      expect(conduitLink.html()).to.contain('href="/myrelpath"');
      expect(conduitLink.html()).to.contain(linkProps.children[0]);
    });

    it('leaves an absolute link unchanged', () => {
      api = ApiHelpers('/my/path/prefix');
      let linkProps = { absolute: "true", to: "http://xkcd.com", children: ["Best Webcomic"] };
      let conduitLink = mount(routerWrap(api.ConduitLink, linkProps));

      expect(conduitLink.find("Link")).to.have.length(1);
      expect(conduitLink.html()).to.contain('href="http://xkcd.com"');
      expect(conduitLink.html()).to.contain(linkProps.children[0]);
    });
  });

  describe('fetch', () => {
    it('adds pathPrefix to a metrics request', () => {
      api = ApiHelpers('/the/path/prefix');
      api.fetch('/resource/foo');

      expect(fetchStub.calledOnce).to.be.true;
      expect(fetchStub.args[0][0]).to.equal('/the/path/prefix/resource/foo');
    });

    it('requests from / when there is no path prefix', () => {
      api = ApiHelpers('');
      api.fetch('/resource/foo');

      expect(fetchStub.calledOnce).to.be.true;
      expect(fetchStub.args[0][0]).to.equal('/resource/foo');
    });

    it('throws an error if response status is not "ok"', () => {
      let errorMessage = "do or do not. there is no try.";
      fetchStub.returnsPromise().resolves({
        ok: false,
        statusText: errorMessage
      });

      api = ApiHelpers('');
      let errorHandler = sinon.spy();

      api.fetch('/resource/foo')
        .catch(errorHandler);

      expect(errorHandler.args[0][0].message).to.equal(errorMessage);
      expect(errorHandler.calledOnce).to.be.true;
    });
  });

  describe('fetchMetrics', () => {
    it('adds pathPrefix and metricsWindow to a metrics request', () => {
      api = ApiHelpers('/the/prefix');
      api.fetchMetrics('/my/path');

      expect(fetchStub.calledOnce).to.be.true;
      expect(fetchStub.args[0][0]).to.equal('/the/prefix/my/path?window=10m');
    });

    it('adds a ?window= if metricsWindow is the only param', () => {
      api.fetchMetrics('/metrics');

      expect(fetchStub.calledOnce).to.be.true;
      expect(fetchStub.args[0][0]).to.equal('/metrics?window=10m');
    });

    it('adds &window= if metricsWindow is not the only param', () => {
      api.fetchMetrics('/metrics?foo=3&bar="me"');

      expect(fetchStub.calledOnce).to.be.true;
      expect(fetchStub.args[0][0]).to.equal('/metrics?foo=3&bar="me"&window=10m');
    });

    it('does not add another &window= if there is already a window param', () => {
      api.fetchMetrics('/metrics?foo=3&window=24h&bar="me"');

      expect(fetchStub.calledOnce).to.be.true;
      expect(fetchStub.args[0][0]).to.equal('/metrics?foo=3&window=24h&bar="me"');
    });
  });

  describe('fetchPods', () => {
    it('fetches the pods from the api', () => {
      api = ApiHelpers("/random/prefix");
      api.fetchPods();

      expect(fetchStub.calledOnce).to.be.true;
      expect(fetchStub.args[0][0]).to.equal('/random/prefix/api/pods');
    });
  });

  describe('urlsForResource', () => {
    it('returns the correct timeseries and metric rollup urls for deployment overviews', () => {
      api = ApiHelpers('/go/my/own/way');
      let deploymentUrls = api.urlsForResource["deployment"].url("myDeploy");

      expect(deploymentUrls.ts).to.equal('/api/metrics?&timeseries=true&target_deploy=myDeploy');
      expect(deploymentUrls.rollup).to.equal('/api/metrics?&target_deploy=myDeploy');
    });

    it('returns the correct timeseries and metric rollup urls for upstream deployments', () => {
      let deploymentUrls = api.urlsForResource["upstream_deployment"].url("farUp");

      expect(deploymentUrls.ts).to.equal('/api/metrics?&aggregation=source_deploy&target_deploy=farUp&timeseries=true');
      expect(deploymentUrls.rollup).to.equal('/api/metrics?&aggregation=source_deploy&target_deploy=farUp');
    });
  });
});
