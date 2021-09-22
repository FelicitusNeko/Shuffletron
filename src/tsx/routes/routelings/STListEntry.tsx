import React, { ChangeEvent, useEffect, useState } from 'react';
import { STList } from '../../interfaces/Shuffletron';

const { port } = window.location

// Test code
//const listList = ['whee', 'games', 'blah'];

interface STListEntryProps {
  setStatus: React.Dispatch<React.SetStateAction<string>>
}
const STListEntry: React.FC<STListEntryProps> = ({ setStatus }) => {
  const [name, setName] = useState('');
  const [delList, setDelList] = useState(0);
  const [listList, setListList] = useState<STList[] | undefined>();
  const [addOp, setAddOp] = useState(false);

  useEffect(() => {
    fetch(`http://localhost:${port}/lists`)
      .then(r => r.json())
      .then(r => setListList(r as STList[]))
      .catch((e: Error) => {
        console.error(e);
        setStatus(`Error: ${e.message}`);
      })
  }, [setStatus]);

  const onAddListNameChange = ({ currentTarget }: ChangeEvent<HTMLInputElement>) => {
    setName(currentTarget.value);
  }

  const onListDeleteChange = ({ currentTarget }: ChangeEvent<HTMLSelectElement>) => {
    setDelList(Number.parseInt(currentTarget.value));
  }

  const onListAdd = () => {
    if (name.length === 0) {
      console.error('No list name specified');
      setStatus(`Error: No list name specified`);
    } else {
      setStatus(`Updating list...`);
      setAddOp(true);
      console.debug('Adding list named:', name);
      fetch(`http://localhost:${port}/lists`, {
        method: 'POST',
        body: JSON.stringify({ name } as Partial<STList>)
      })
        .then(r => r.json())
        .then(r => {
          if (r.err) throw new Error(r.err);
          else {
            setAddOp(false);
            const newList = r as STList
            if (listList) setListList(listList.slice(0).concat(newList));
            setStatus(`List created: ${newList.name}`);
            setName('');
          }
        })
        .catch((e: Error) => {
          setAddOp(false);
          console.error(e);
          setStatus(`Error: ${e.message}`);
        })
    }
  }

  const onListDelete = () => {
    if (!listList) return;
    const delListName = listList.reduce((r, i) => i.id === delList ? i.name : r, '')
    if (delListName === '') setStatus('Error: Cannot find list to delete');
    else {
      console.debug('Deleting list named:', delListName);
      fetch(`http://localhost:${port}/lists/${delList}`, {
        method: 'DELETE'
      })
        .then(r => {
          setListList(listList.filter(i => i.id !== delList))
          setDelList(0);
          setStatus(`Deleted list: ${delListName}`);
        })
        .catch((e: Error) => {
          console.error(e);
          setStatus(`Error: ${e.message}`);
        })
    }
  }

  return <>
    <h3>Lists</h3>
    <p>
      <input type='text'
        placeholder='Add new list'
        value={name}
        onChange={onAddListNameChange}
        disabled={addOp}
        required
      /> <button disabled={addOp} onClick={onListAdd}>Add</button>
    </p>
    <p>
      Delete list: <select
        value={delList}
        onChange={onListDeleteChange}
        disabled={!listList || listList.length === 0}
      >
        {listList
          ? listList.map(i => <option key={`dellist-${i.id}`} value={i.id} >{i.name}</option>)
          : <option>Loading...</option>
        }
      </select> <button disabled={!listList || listList.length === 0} onClick={onListDelete}>
        Delete
      </button>
    </p>
  </>
}

export default STListEntry