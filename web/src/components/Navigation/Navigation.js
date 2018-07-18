import React from 'react';
import { BrowserRouter as Router, Route, Link } from 'react-router-dom';
import { Home, About } from '../../pages';

const Navigation = () => (
  <Router>
    <div>
      <ul>
        <li>
          <Link to="/">Home</Link>
        </li>
        <li>
          <Link to="/about">About</Link>
        </li>
      </ul>

      <Route exact path="/" component={Home} />
      <Route path="/about" component={About} />
    </div>
  </Router>
);

export default Navigation;
