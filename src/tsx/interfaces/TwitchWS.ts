export interface TwitchWSMsg {
  id: string;
  displayName: string;
  displayCol: string;
  channel: string;
  msg: string;
  time: number;
  emotes: TwitchWSMsgEmote[]
}

export interface TwitchWSMsgEmote {
  name: string;
  id: string;
}
