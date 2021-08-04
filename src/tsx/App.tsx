import React from 'react';
import { Switch, Route, Link } from 'react-router-dom'
import Chat from './routes/Chat';
//import '../css/App.css';

function App() {
  return (
    <div className='App'>
      <Switch>
        <Route path='/chatoverlay'>
          <Chat />
        </Route>
        <Route>
          <Link to='/chatoverlay'>Chat overlay</Link>
        </Route>
      </Switch>
    </div>
  );
}

export default App;
