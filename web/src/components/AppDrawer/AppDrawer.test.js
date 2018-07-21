import React from 'react';
import ReactDOM from 'react-dom';
import AppDrawer from './AppDrawer';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(<AppDrawer />, div);
  ReactDOM.unmountComponentAtNode(div);
});
