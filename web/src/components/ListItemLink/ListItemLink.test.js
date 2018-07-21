import React from 'react';
import ReactDOM from 'react-dom';
import { MemoryRouter } from 'react-router';
import ListItemLink from './ListItemLink';
import {
  Code as CodeIcon,
} from '@material-ui/icons';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(
    <MemoryRouter>
      <ListItemLink icon={<CodeIcon />} primary="Homepage" to="/" />
    </MemoryRouter>, div);
  ReactDOM.unmountComponentAtNode(div);
});
