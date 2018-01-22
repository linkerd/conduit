import Adapter from 'enzyme-adapter-react-16';
import DeploymentsList from '../js/components/DeploymentsList.jsx';
import Enzyme from 'enzyme';
import { expect } from 'chai';
import { mount } from 'enzyme';
import podFixtures from './fixtures/pods.json';
import { routerWrap } from './testHelpers.jsx';
import sinon from 'sinon';
import sinonStubPromise from 'sinon-stub-promise';

Enzyme.configure({ adapter: new Adapter() });
sinonStubPromise(sinon);

describe('DeploymentsList', () => {
  let component, fetchStub;

  function withPromise(fn) {
    return component.find("DeploymentsList").instance().serverPromise.then(fn);
  }

  beforeEach(() => {
    fetchStub = sinon.stub(window, 'fetch');
  });

  afterEach(() => {
    component = null;
    window.fetch.restore();
  });

  it('renders the spinner before metrics are loaded', () => {
    fetchStub.returnsPromise().resolves({ ok: true });
    component = mount(routerWrap(DeploymentsList));

    return withPromise(() => {
      expect(component.find("DeploymentsList")).to.have.length(1);
      expect(component.find("ConduitSpinner")).to.have.length(1);
      expect(component.find("CallToAction")).to.have.length(0);
    });
  });

  it('renders a call to action if no metrics are received', () => {
    fetchStub.returnsPromise().resolves({
      ok: true,
      json: () => Promise.resolve({ metrics: [] })
    });
    component = mount(routerWrap(DeploymentsList));

    return withPromise(() => {
      component.update();
      expect(component.find("DeploymentsList").length).to.equal(1);
      expect(component.find("ConduitSpinner").length).to.equal(0);
      expect(component.find("CallToAction").length).to.equal(1);
    });
  });

  it('renders the deployments page if pod data is received', () => {
    fetchStub.returnsPromise().resolves({
      ok: true,
      json: () => Promise.resolve({ metrics: [], pods: podFixtures.pods })
    });
    component = mount(routerWrap(DeploymentsList));

    return withPromise(() => {
      component.update();
      expect(component.find("DeploymentsList").length).to.equal(1);
      expect(component.find("ConduitSpinner").length).to.equal(0);
      expect(component.find("CallToAction").length).to.equal(0);
      expect(component.find("TabbedMetricsTable").length).to.equal(1);
    });
  });
});
