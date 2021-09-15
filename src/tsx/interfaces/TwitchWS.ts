export interface TwitchWSMsg {
  msgType: TwitchWSMsgType;
  id: string;
  displayName: string;
  displayCol: string;
  channel: string;
  msg: string;
  time: number;
  emotes: TwitchWSMsgEmote[];
}

export interface TwitchWSMsgEmote {
  name: string;
  id: string;
}

export enum TwitchWSMsgType {
  Unknown,
  Message,
  Action,
  Delete
}