import React, { ChangeEvent, useEffect, useState } from 'react';
import { STGame, STList } from '../../interfaces/Shuffletron';

const MinWeight = 1;
const MaxWeight = 25000;

const { port } = window.location;

interface STGameEntryProps {
  setStatus: React.Dispatch<React.SetStateAction<string>>
}
const STGameEntry: React.FC<STGameEntryProps> = ({ setStatus }) => {
  const [curList, setCurList] = useState(0);
  const [name, setName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  const [weight, setWeight] = useState(1);
  const [statusPlayed, setStatusPlayed] = useState(false);
  const [statusMultiplayer, setStatusMultiplayer] = useState(false);
  const [delGame, setDelGame] = useState(0);

  const [listList, setListList] = useState<STList[] | undefined>();
  const [gameList, setGameList] = useState<STGame[] | undefined>();

  const [activeOp, setActiveOp] = useState(false);

  useEffect(() => {
    console.debug('Loading lists');

    fetch(`http://localhost:${port}/lists`)
      .then(r => r.json())
      .then(r => {
        const newList = r as STList[];
        setListList(newList);
        if (newList.length > 0) setCurList(newList[0].id);
      })
      .catch((e: Error) => {
        console.error(e);
        setStatus(`Error getting lists: ${e.message}`);
      });
  }, [setStatus]);

  useEffect(() => {
    if (listList) {
      const curListName = listList.reduce((r, i) => i.id === curList ? i.name : r, '');
      if (curListName === '') {
        setGameList([]);
        return;
      }
      console.debug(`Loading game list for ${curListName}`);

      fetch(`http://localhost:${port}/games/byList/${curList}`)
        .then(r => r.json())
        .then(r => setGameList(r as STGame[]))
        .catch((e: Error) => {
          console.error(e);
          setStatus(`Error getting games: ${e.message}`);
        });
    }
  }, [listList, curList, setStatus])

  const onCurListChange = ({ currentTarget }: ChangeEvent<HTMLSelectElement>) => {
    const newCurList = Number.parseInt(currentTarget.value);
    if (!isNaN(newCurList)) setCurList(newCurList);
  }

  const onGameAddNameChange = ({ currentTarget }: ChangeEvent<HTMLInputElement>) => {
    setName(currentTarget.value);
  }

  const onGameAddDisplayNameChange = ({ currentTarget }: ChangeEvent<HTMLInputElement>) => {
    setDisplayName(currentTarget.value.substring(0, 20));
  }

  const onGameAddDescriptionChange = ({ currentTarget }: ChangeEvent<HTMLTextAreaElement>) => {
    setDescription(currentTarget.value);
  }

  const onGameAddWeightChange = ({ currentTarget }: ChangeEvent<HTMLInputElement>) => {
    const newWeight = Number.parseInt(currentTarget.value);
    setWeight(isNaN(newWeight) ? 1 : Math.min(MaxWeight, Math.max(MinWeight, newWeight)));
  }

  const onGameAddPlayedChange = ({ currentTarget }: ChangeEvent<HTMLInputElement>) => {
    setStatusPlayed(currentTarget.checked);
  }

  const onGameAddMultiplayerChange = ({ currentTarget }: ChangeEvent<HTMLInputElement>) => {
    setStatusMultiplayer(currentTarget.checked);
  }

  const onGameAdd = () => {
    if (name.length === 0) {
      console.error('No game name specified');
    } else if (weight < MinWeight || weight > MaxWeight) {
      console.error(`Invalid weight value; must be ${MinWeight}-${MaxWeight}`);
    } else {
      setStatus(`Adding game...`);
      console.debug('Adding game named:', name);
      setActiveOp(true);

      const newGame: Partial<STGame> = {
        listId: curList,
        name,
        weight
      };
      if (displayName) newGame.displayName = displayName;
      if (description) newGame.description = description;
      newGame.status = 0;
      if (statusPlayed) newGame.status |= 1;
      if (statusMultiplayer) newGame.status |= 2;

      fetch(`http://localhost:${port}/games`, {
        method: 'POST',
        body: JSON.stringify(newGame)
      })
        .then(r => r.json())
        .then(r => {
          if (r.err) throw new Error(r.err);
          else {
            const newGame = r as STGame
            if (gameList) setGameList(gameList.slice(0).concat(newGame));
            setStatus(`Game added: ${newGame.name}`);
            onGameAddClear();
          }
        })
        .catch((e: Error) => {
          setActiveOp(false);
          console.error(e);
          setStatus(`Error: ${e.message}`);
        });
    }
  }

  const onGameAddClear = () => {
    setName('');
    setDisplayName('');
    setDescription('');
    setWeight(1);
    setStatusPlayed(false);
    setStatusMultiplayer(false);
    setActiveOp(false);
  }

  const onGameDeleteChange = ({ currentTarget }: ChangeEvent<HTMLSelectElement>) => {
    setDelGame(Number.parseInt(currentTarget.value));
  }

  const onGameDelete = () => {
    if (gameList) {
      const delGameName = gameList.reduce((r, i) => i.id === delGame ? i.name : r, '');
      if (delGameName === '') setStatus('Error: Cannot find list to delete');
      else {
        setStatus('Deleting list...');
        console.debug('Deleting game named:', delGameName);
        setActiveOp(true);
        fetch(`http://localhost:${port}/games/${delGame}`, {
          method: 'DELETE'
        })
          .then(r => {
            setActiveOp(false);
            setListList(gameList.filter(i => i.id !== delGame))
            setDelGame(0);
            setStatus(`Deleted game: ${delGameName}`);
          })
          .catch((e: Error) => {
            setActiveOp(false);
            console.error(e);
            setStatus(`Error: ${e.message}`);
          });
      }
    }
  }

  return <>
    <h3>Games</h3>
    <p>
      List: <select onChange={onCurListChange}>
        {listList
          ? listList.map(i => <option key={`worklist-${i.id}`} value={i.id}>{i.name}</option>)
          : <option>Loading...</option>
        }
      </select>
    </p>
    <fieldset disabled={activeOp || !listList || listList.length === 0}>
      <legend>Add new game</legend>
      <p>
        <input type='text'
          placeholder='Game name'
          required
          value={name}
          onChange={onGameAddNameChange}
        />
      </p>
      <p>
        <input type='text'
          placeholder='Display name (opt)'
          maxLength={20}
          value={displayName}
          onChange={onGameAddDisplayNameChange}
        />
      </p>
      <p>Game will display as: <span className='digifont stDigiDisplay'>
        <span className='stDigiBackground'>@@@@@@@@@@@@@@@@@@@@</span>
        <span className='stDigiForeground'>{displayName.length > 0 ? displayName : name.substring(0, 20)}</span>
      </span></p>
      <p>
        <textarea
          placeholder='Description'
          value={description}
          rows={4} cols={60}
          onChange={onGameAddDescriptionChange}
        />
      </p>
      <p>
        Weight: <input type='number'
          value={weight}
          min={1}
          max={25000}
          onChange={onGameAddWeightChange}
        />
      </p>
      <p>Status:</p>
      <ul>
      <li>
          <label>
            <input type='checkbox'
              checked={statusPlayed}
              onChange={onGameAddPlayedChange}
            /> Played
          </label>
        </li>
        <li>
          <label>
            <input type='checkbox'
              checked={statusMultiplayer}
              onChange={onGameAddMultiplayerChange}
            /> Played
          </label>
        </li>
      </ul>
      <p>
        <button onClick={onGameAdd}>Add</button>&nbsp;
        <button onClick={onGameAddClear}>Clear</button>
      </p>
    </fieldset>
    <fieldset disabled={activeOp || !listList || !gameList || gameList.length === 0}>
      <legend>Delete game</legend>
      <p>
        <select value={delGame} onChange={onGameDeleteChange}>
          {gameList
            ? gameList.map(i => <option key={`delgame-${i.id}`} value={i.id} >{i.name}</option>)
            : <option>Loading...</option>
          }
        </select> <button onClick={onGameDelete}>Delete</button>
      </p>
    </fieldset>
  </>;
}

export default STGameEntry;