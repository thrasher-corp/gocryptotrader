import React from 'react';
import ReactDOM from 'react-dom';
import Settings from './Settings';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(<Settings />, div);
  ReactDOM.unmountComponentAtNode(div);
});
