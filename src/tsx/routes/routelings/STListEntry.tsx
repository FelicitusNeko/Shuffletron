import React, { ChangeEvent, useState } from 'react';

// Test code
const listList = ['whee', 'games', 'blah'];

interface STListEntryProps {
  setStatus: React.Dispatch<React.SetStateAction<string>>
}
const STListEntry: React.FC<STListEntryProps> = ({ setStatus }) => {
  const [name, setName] = useState('');
  const [delList, setDelList] = useState(0);

  const onAddListNameChange = ({ currentTarget }: ChangeEvent<HTMLInputElement>) => {
    setName(currentTarget.value);
  }

  const onListDeleteChange = ({ currentTarget }: ChangeEvent<HTMLSelectElement>) => {
    setDelList(currentTarget.selectedIndex);
  }

  const onListAdd = () => {
    if (name.length === 0) {
      console.error('No list name specified');
    } else {
      console.debug('Adding list named:', name);
    }
  }

  const onListDelete = () => {
    console.debug('Deleting list named:', listList[delList]);
  }

  return <>
    <h3>Lists</h3>
    <p>
      <input type='text'
        placeholder='Add new list'
        value={name}
        onChange={onAddListNameChange}
        required
      /> <button onClick={onListAdd}>Add</button>
    </p>
    <p>
      Delete list: <select value={delList} onChange={onListDeleteChange}>
        {listList.map((i, x) => <option key={`dellist-${x}`} value={x} >{i}</option>)}
      </select> <button onClick={onListDelete}>Delete</button>
    </p>
  </>
}

export default STListEntry