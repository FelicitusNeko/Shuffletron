import React, { ReactNode, useEffect, useState } from 'react';
import Sockette from 'sockette';

interface ChatItemProps {
  children?: ReactNode
}
const ChatItem: React.FC<ChatItemProps> = ({ children }) => {
  console.log('trying to render chatitem')
  return <p>
    {children || ''}
  </p>
}

const Chat: React.FC = () => {
  const [ws, setWs] = useState<Sockette>();
  const [msgList, setMsgList] = useState<ReactNode[]>([]);

  /*const ws = new Sockette('ws://localhost:42069/ws', {
    timeout: 5000,
    maxAttempts: 10,
    onopen: e => console.log('Connected!', e),
    onmessage: e => onMessage(e),
    onreconnect: e => console.log('Reconnecting...', e),
    onmaximum: e => console.log('Stop Attempting!', e),
    onclose: e => console.log('Closed!', e),
    onerror: e => console.log('Error:', e)
  });*/

  useEffect(() => {
    const onMessage = (e: MessageEvent<any>) => {
      console.log('Received:', e)
      const msg = <ChatItem key={Date.now().toString()}>{e.data}</ChatItem>
      msgList.push(msg);
      setMsgList(msgList);
      setTimeout(() => setMsgList(msgList.filter(i => i !== msg)), 10000);
    }

    if (!ws) setWs(new Sockette('ws://localhost:42069/ws', {
      timeout: 5000,
      maxAttempts: 10,
      onopen: e => console.log('Connected!', e),
      onmessage: e => onMessage(e),
      onreconnect: e => console.log('Reconnecting...', e),
      onmaximum: e => console.log('Stop Attempting!', e),
      onclose: e => console.log('Closed!', e),
      onerror: e => console.log('Error:', e)
    }));
    /*else return () => {
      ws.close();
      setWs(undefined);
    }*/
  }, [ws, msgList]);


  return <>
    <ChatItem>Hello World! This is the chat overlay.</ChatItem>
    {msgList}
  </>
}

export default Chat;