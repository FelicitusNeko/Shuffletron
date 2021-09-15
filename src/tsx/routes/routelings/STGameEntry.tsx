import React, { ChangeEvent, useState } from 'react';

// Test code
const listList = ['whee', 'games', 'blah'];
const gameList = ['Abadox', 'ActRaiser', 'CrossCode'];

const MinWeight = 1;
const MaxWeight = 25000;

interface STGameEntryProps {
  setStatus: React.Dispatch<React.SetStateAction<string>>
}
const STGameEntry: React.FC<STGameEntryProps> = ({ setStatus }) => {
  const [name, setName] = useState('');
  const [displayName, setDisplayName] = useState('');
  const [description, setDescription] = useState('');
  const [weight, setWeight] = useState(1);
  const [statusPlayed, setStatusPlayed] = useState(false);
  const [delGame, setDelGame] = useState(0);

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

  const onGameAdd = () => {
    if (name.length === 0) {
      console.error('No game name specified');
    } else if (weight < MinWeight || weight > MaxWeight) {
      console.error(`Invalid weight value; must be ${MinWeight}-${MaxWeight}`);
    } else {
      console.debug('Adding game named:', name);
      // if successful then
      onGameAddClear();
    }
  }

  const onGameAddClear = () => {
    setName('');
    setDisplayName('');
    setDescription('');
    setWeight(1);
    setStatusPlayed(false);
  }

  const onGameDeleteChange = ({ currentTarget }: ChangeEvent<HTMLSelectElement>) => {
    setDelGame(currentTarget.selectedIndex);
  }

  const onGameDelete = () => {
    console.debug('Deleting game named:', gameList[delGame]);
  }


  return <>
    <h3>Games</h3>
    <p>
      List: <select>
        {listList.map((i, x) => <option key={`worklist-${x}`} value={x} >{i}</option>)}
      </select>
    </p>
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
    <p>Game will display as: <span className='digifont'>
      {displayName.length > 0 ? displayName : name.substring(0, 20)}
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
    </ul>
    <p>
      <button onClick={onGameAdd}>Add</button>&nbsp;
      <button onClick={onGameAddClear}>Clear</button>
    </p>
    <p>
      Delete game: <select value={delGame} onChange={onGameDeleteChange}>
        {gameList.map((i, x) => <option key={`delgame-${x}`} value={x} >{i}</option>)}
      </select> <button onClick={onGameDelete}>Delete</button>
    </p>
  </>;
}

export default STGameEntry