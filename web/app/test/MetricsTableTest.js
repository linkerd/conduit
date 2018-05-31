import Adapter from 'enzyme-adapter-react-16';
import ApiHelpers from '../js/components/util/ApiHelpers.jsx';
import BaseTable from '../js/components/BaseTable.jsx';
import { expect } from 'chai';
import { MetricsTableBase } from '../js/components/MetricsTable.jsx';
import React from 'react';
import Enzyme, { shallow } from 'enzyme';

Enzyme.configure({ adapter: new Adapter() });

describe('Tests for <MetricsTableBase>', () => {
  const defaultProps = {
    api: ApiHelpers(''),
  };

  it('renders the table with all columns', () => {
    const component = shallow(
      <MetricsTableBase
        {...defaultProps}
        metrics={[{
          name: 'web',
          namespace: 'default',
          totalRequests: 0,
        }]}
        resource="deployment" />
    );

    const table = component.find(BaseTable);

    expect(table).to.have.length(1);
    expect(table.props().dataSource).to.have.length(1);
    expect(table.props().columns).to.have.length(8);
  });

  it('omits the namespace column for the namespace resource', () => {
    const component = shallow(
      <MetricsTableBase
        {...defaultProps}
        metrics={[]}
        resource="namespace" />
    );

    const table = component.find(BaseTable);

    expect(table).to.have.length(1);
    expect(table.props().columns).to.have.length(7);
  });

  it('omits the namespace column when showNamespaceColumn is false', () => {
    const component = shallow(
      <MetricsTableBase
        {...defaultProps}
        metrics={[]}
        resource="deployment"
        showNamespaceColumn={false} />
    );

    const table = component.find(BaseTable);

    expect(table).to.have.length(1);
    expect(table.props().columns).to.have.length(7);
  });

});
