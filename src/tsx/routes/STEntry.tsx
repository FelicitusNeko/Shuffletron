import React, { useState } from 'react';
import { Tab, TabList, TabPanel, Tabs } from 'react-tabs';

import STGameEntry from './routelings/STGameEntry';
import STListEntry from './routelings/STListEntry';

import 'react-tabs/style/react-tabs.css';
import '../../css/STEntry.css';

const STEntry: React.FC = (props) => {
  const [status, setStatus] = useState('Test status');

  const onTabSelect = () => {
    setStatus('');
  }

  return <div className='shuffletron stEntry'>
    <Tabs defaultIndex={1} onSelect={onTabSelect}>
      <TabList>
        <Tab>Lists</Tab>
        <Tab>Games</Tab>
      </TabList>

      <div id='stStatus'>{status}</div>

      <TabPanel>
        <STListEntry setStatus={setStatus} />
      </TabPanel>

      <TabPanel>
        <STGameEntry setStatus={setStatus} />
      </TabPanel>
    </Tabs>
  </div>;
}

export default STEntry;