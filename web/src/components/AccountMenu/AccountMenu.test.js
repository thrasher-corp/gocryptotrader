import React from 'react';
import ReactDOM from 'react-dom';
import AccountMenu from './AccountMenu';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(<AccountMenu />, div);
  ReactDOM.unmountComponentAtNode(div);
});
