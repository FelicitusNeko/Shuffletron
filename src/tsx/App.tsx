import React from 'react';
import { Routes, Route, Link } from 'react-router-dom'

import Chat from './routes/Chat';
import STDisplay from './routes/STDisplay';
import STEntry from './routes/STEntry';

const App: React.FC = () => {
  return (
    <div className='App'>
      <Routes>
        <Route path='/chatoverlay' element={<Chat />} />
        <Route path='/st-entry' element={<STEntry />} />
        <Route path='/st-display' element={<STDisplay />} />
        <Route path='*' element={
          <ul>
            <li><Link to='/chatoverlay'>Chat overlay</Link></li>
            <li><Link to='/st-entry'>Shuffletron Entry</Link></li>
            <li><Link to='/st-display'>Shuffletron Display</Link></li>
          </ul>
        } />
      </Routes>
    </div>
  );
}

export default App;
