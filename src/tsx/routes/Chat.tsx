import React, { ReactNode } from 'react';
import Sockette from 'sockette';
import '../../css/Chat.css';
import { DateTime } from 'luxon';

interface TwitchWSMsg {
  id: string;
  displayName: string;
  displayCol: string;
  channel: string;
  msg: string;
  time: number;
}

interface ChatItemProps {
  displayName?: string;
  displayCol?: string;
  channel?: string;
  time?: DateTime;
  children?: ReactNode
}
const ChatItem: React.FC<ChatItemProps> = ({ displayName, displayCol, time, channel, children }) => {
  const nameStyle: React.CSSProperties = {
    fontWeight: 'bold'
  };
  if (displayCol) nameStyle.color = displayCol;

  let chatTime = time ? <span className='chatTime'>
    {`${time.toLocaleString(DateTime.TIME_24_SIMPLE)}`}
  </span> : null;
  let chatChannel = channel ? <span className='channelName'>
    [{channel.slice(0, 3).toUpperCase()}]&nbsp;
  </span> : null;
  let chatUser = displayName ? <span className='chatName' style={nameStyle}>
    {displayName}:&nbsp;
  </span> : null;

  return <p>
    {chatTime}
    {chatChannel}
    {chatUser}
    {children || ''}
  </p>
}

interface ChatProps {

}
interface ChatState {
  ws: Sockette;
  msgList: ReactNode[];
}
export default class Chat extends React.Component<ChatProps, ChatState> {
  /*constructor(props: ChatProps) {
    super(props);
  }*/

  componentDidMount() {
    this.setState({
      ws: new Sockette('ws://localhost:42069/ws', {
        timeout: 5000,
        maxAttempts: 10,
        onopen: e => console.log('Connected!', e),
        onmessage: e => this.onMessage(e),
        onreconnect: e => console.log('Reconnecting...', e),
        onmaximum: e => console.log('Stop Attempting!', e),
        onclose: e => console.log('Closed!', e),
        onerror: e => console.error('Error:', e)
      }),
      msgList: []
    });
  }

  componentWillUnmount() {
    this.state.ws.close();
  }

  onMessage = (e: MessageEvent<any>) => {
    const { msgList } = this.state;
    const newMsgList = msgList.slice();
    const inMsg = JSON.parse(e.data) as TwitchWSMsg

    console.log('Received:', e);
    console.debug('Parsed content:', inMsg);
    const msg = <ChatItem
      key={inMsg.id}
      displayName={inMsg.displayName}
      displayCol={inMsg.displayCol}
      //time={new Date(inMsg.time * 1000)}
      channel={inMsg.channel}
      time={DateTime.fromMillis(inMsg.time * 1000).toLocal()}
    >
      {inMsg.msg}
    </ChatItem>;

    console.debug('Output chatitem:', msg);
    newMsgList.push(msg);
    this.setState({ msgList: newMsgList });

    setTimeout(() => {
      this.setState({
        msgList: this.state.msgList.filter(i => i !== msg)
      })
    }, 30000);
  }

  render() {
    if (!this.state) return null;
    const { msgList } = this.state;

    return <div id='chat'>
      <ChatItem>Hello World! This is the chat overlay.</ChatItem>
      {msgList}
    </div>
  }
}
