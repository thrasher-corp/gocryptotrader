import React from 'react';
import ReactDOM from 'react-dom';
import DonationAddress from './DonationAddress';

it('renders without crashing', () => {
  const div = document.createElement('div');
  ReactDOM.render(<DonationAddress address="abcdefgh" currency="BTC" />, div);
  ReactDOM.unmountComponentAtNode(div);
});
