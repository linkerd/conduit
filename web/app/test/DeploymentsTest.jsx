import Deployments from '../js/components/Deployments.jsx';
import { expect } from 'chai';
import { mount } from 'enzyme';
import podFixtures from './fixtures/pods.json';
import { routerWrap } from "./testHelpers.jsx";
import sinon from 'sinon';
import sinonStubPromise from 'sinon-stub-promise';

sinonStubPromise(sinon);

describe('Deployments', () => {
  let component, fetchStub;

  function withPromise(fn) {
    return component.find("Deployments").get(0).serverPromise.then(fn);
  }

  beforeEach(() => {
    fetchStub = sinon.stub(window, 'fetch');
  });

  afterEach(() => {
    window.fetch.restore();
  });

  it('renders the spinner before metrics are loaded', () => {
    fetchStub.returnsPromise().resolves({
      json: () => Promise.resolve({ metrics: [] })
    });
    component = mount(routerWrap(Deployments));

    expect(component.find("Deployments")).to.have.length(1);
    expect(component.find("ConduitSpinner")).to.have.length(1);
    expect(component.find("CallToAction")).to.have.length(0);
  });

  it('renders a call to action if no metrics are received', () => {
    fetchStub.returnsPromise().resolves({
      json: () => Promise.resolve({ metrics: [] })
    });
    component = mount(routerWrap(Deployments));

    return withPromise(() => {
      expect(component.find("Deployments")).to.have.length(1);
      expect(component.find("ConduitSpinner")).to.have.length(0);
      expect(component.find("CallToAction")).to.have.length(1);
    });
  });

  it('renders the deployments page if pod data is received', () => {
    fetchStub.returnsPromise().resolves({
      json: () => Promise.resolve({ metrics: [], pods: podFixtures.pods })
    });
    component = mount(routerWrap(Deployments));

    return withPromise(() => {
      expect(component.find("Deployments")).to.have.length(1);
      expect(component.find("ConduitSpinner")).to.have.length(0);
      expect(component.find("CallToAction")).to.have.length(0);
      expect(component.find("TabbedMetricsTable")).to.have.length(1);
    });
  });
});
