import React, { useEffect, useState } from 'react';
import { STList, STShuffleResult } from '../interfaces/Shuffletron';

import '../../css/Shuffletron.css';

const { port } = window.location;

const STDisplay: React.FC = () => {
  const [listList, setListList] = useState<STList[] | null>(null);
  const [curList, setCurList] = useState<number | undefined>();
  const [result, setResult] = useState<STShuffleResult | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [shuffleAnim, setShuffleAnim] = useState(0);
  const [activeOp, setActiveOp] = useState(false);

  useEffect(() => {
    fetch(`http://localhost:${port}/lists`)
      .then(r => r.json())
      .then(r => setListList(r as STList[]))
      .catch((e: Error) => {
        console.error(e);
        setError('Load list err');
      })
  }, []);

  useEffect(() => {
    setCurList((listList && listList.length > 0) ? listList[0].id : undefined);
  }, [listList]);

  useEffect(() => {
    const DoAnim = () => {
      if (!result) return;
      const sel = Math.floor(Math.random() * result.animContent.length);
      setError(result.animContent[sel]);
      setShuffleAnim(shuffleAnim - 1);
    }
    if (!result) {
      if (shuffleAnim > 0) setShuffleAnim(0);
    }
    else if (shuffleAnim > 0) setTimeout(DoAnim, 100);
    else {
      setActiveOp(false);
      setError(null);
    }
  }, [shuffleAnim, result]);

  const onClear = () => {
    setResult(null);
    setError(null);
  }

  const onShuffle = () => {
    if (curList === undefined) {
      console.error('No list selected');
      setError('NO LIST ERR');
    } else {
      setError('WAIT...')
      console.debug('Retrieving shuffle set');
      setActiveOp(true);
      fetch(`http://localhost:${port}/shuffle/${curList}`)
        .then(r => r.json())
        .then(r => {
          if (r.err) throw new Error(r.err);
          else {
            setError(null);
            setResult(r as STShuffleResult);
            setShuffleAnim(20);
          }
        })
        .catch((e: Error) => {
          setActiveOp(false);
          console.error(e);
          setError('SHUFFLE ERR');
        });
    }
  }

  const onMark = () => {
    if (!result) {
      console.error('No result to mark done');
      setError('NO RESULT ERR');
      setTimeout(() => setError(null), 500);
    } else {
      setError('WAIT...')
      console.debug(`Marking ${result.game.name} as played...`);
      result.game.status |= 1;
      setActiveOp(true);
      fetch(`http://localhost:${port}/games/${result.game.id}`, {
        method: 'PUT',
        body: JSON.stringify(result.game)
      })
        .then(r => r.json())
        .then(r => {
          if (r.err) throw new Error(r.err);
          else {
            setActiveOp(false);
            setError(null);
            setResult(null);
          }
        })
        .catch((e: Error) => {
          setActiveOp(false);
          console.error(e);
          setError('MARK DONE ERR');
          setTimeout(() => setError(null), 500);
        });
    }
  }

  return <div className='shuffletron'>
    <div className='stDisplay'>
      <div className='stDisplayInner'>
        <div className='stTitle'>Shuffletron</div>
        <div className='stFlex'>
          <span className='digifont stDigiDisplay'>
            <span className='stDigiBackground'>@@@@@@@@@@@@@@@@@@@@</span>
            <span className='stDigiForeground'>{(
              error ?? (result
                ? (result.game.displayName ?? result.game.name)
                : 'KM-Shuffletron 1000')
            ).substr(0, 20)}</span>
          </span>
          <select disabled={activeOp} value={curList}>
            <option value='noPick'>Select list</option>
            {listList
              ? listList.map(i => <option key={`shufsel-${i.id}`} value={i.id}>{i.name}</option>)
              : null}
          </select>
          <button disabled={activeOp} onClick={onClear}>Clear</button>
          <button disabled={activeOp} onClick={onShuffle}>Shuffle!</button>
          <button disabled={activeOp || !result} onClick={onMark}>Mark Done</button>
        </div>
      </div>
    </div>
  </div>;
}

export default STDisplay