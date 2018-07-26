import React from 'react';
import ReactDOM from 'react-dom';
import EmptyState from './EmptyState';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(<EmptyState isLoading />, div);
  ReactDOM.unmountComponentAtNode(div);
});
