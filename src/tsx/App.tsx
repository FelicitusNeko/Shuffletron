import React from 'react';
import { Switch, Route, Link } from 'react-router-dom'

import Chat from './routes/Chat';
import STDisplay from './routes/STDisplay';
import STEntry from './routes/STEntry';

const App: React.FC = () => {
  return (
    <div className='App'>
      <Switch>
        <Route path='/chatoverlay' component={Chat} />
        <Route path='/st-entry' component={STEntry} />
        <Route path='/st-display' component={STDisplay} />
        <Route>
          <ul>
            <li><Link to='/chatoverlay'>Chat overlay</Link></li>
            <li><Link to='/st-entry'>Shuffletron Entry</Link></li>
            <li><Link to='/st-display'>Shuffletron Display</Link></li>
          </ul>
        </Route>
      </Switch>
    </div>
  );
}

export default App;
