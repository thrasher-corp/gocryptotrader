import React from 'react';
import ReactDOM from 'react-dom';
import WebserverSettings from './WebserverSettings';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(<WebserverSettings data={{}} />, div);
  ReactDOM.unmountComponentAtNode(div);
});
