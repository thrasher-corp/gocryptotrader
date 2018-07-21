import React from 'react';
import ReactDOM from 'react-dom';
import { MemoryRouter } from 'react-router';
import MenuItems from './MenuItems';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(
    <MemoryRouter>
      <MenuItems />
    </MemoryRouter>, div);
  ReactDOM.unmountComponentAtNode(div);
});
