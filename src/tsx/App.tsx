import React from 'react';
import { Switch, Route } from 'react-router-dom'
import Chat from './routes/Chat';
//import '../css/App.css';

function App() {
  return (
    <div className='App'>
      <Switch>
        <Route path='/chatoverlay'>
          <Chat />
        </Route>
      </Switch>
    </div>
  );
}

export default App;
