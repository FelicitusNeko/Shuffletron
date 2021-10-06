import React, { useEffect, useState } from 'react';
import { STList, STShuffleResult } from '../interfaces/Shuffletron';

import '../../css/Shuffletron.css';

const { port } = window.location;

const STDisplay: React.FC = () => {
  const [listList, setListList] = useState<STList[] | null>(null);
  const [result, setResult] = useState<STShuffleResult | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    fetch(`http://localhost:${port}/lists`)
      .then(r => r.json())
      .then(r => setListList(r as STList[]))
      .catch((e: Error) => {
        console.error(e);
        setError('Load list err');
      })
  }, []);

  return <div className="shuffletron">
    <div className="stDisplay">
      <div className="stDisplayInner">
        <div className="stTitle">Shuffletron</div>
        <div style={{ textAlign: 'center' }}>
          <span className='digifont stDigiDisplay'>
            <span className='stDigiBackground'>@@@@@@@@@@@@@@@@@@@@</span>
            <span className='stDigiForeground'>{(error ?? 'KM-Shuffletron 1000').substr(0, 20)}</span>
          </span>
        </div>
        <div>
          <select>
            <option>Select list</option>
          </select>
        </div>
        <div>
          <button>Clear</button>
          <button>Shuffle!</button>
        </div>
      </div>
    </div>
  </div>;
}

export default STDisplay