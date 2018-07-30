import React from 'react';
import ReactDOM from 'react-dom';
import SecretInput from './SecretInput';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(
    <SecretInput id="topSecret" label="Secret" value="cannottellyou" />,
    div
  );
  ReactDOM.unmountComponentAtNode(div);
});
