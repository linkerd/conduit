import _ from 'lodash';
import { ApiHelpers } from '../js/components/util/ApiHelpers.jsx';
import { createMemoryHistory } from 'history';
import React from 'react';
import { Route, Router } from 'react-router';

const componentDefaultProps = { api: ApiHelpers("") };

export function routerWrap(Component, extraProps={}, route="/", currentLoc="/") {
  const createElement = (ComponentToWrap, props) => <ComponentToWrap {...(_.merge({}, componentDefaultProps, props, extraProps))} />;
  return (
    <Router history={createMemoryHistory(currentLoc)} createElement={createElement}>
      <Route path={route} render={props => createElement(Component, props)} />
    </Router>
  );
}
