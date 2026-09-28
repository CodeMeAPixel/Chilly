export type LoopMode = "none" | "track" | "queue";

export type Track = {
  title: string;
  author: string;
  uri?: string;
  artwork_url?: string;
  source: string;
  identifier: string;
  length_ms: number;
  is_stream: boolean;
  requester_id?: string;
  playlist_name?: string;
  radio_station?: string;
};

export type PlayerState = {
  guild_id: string;
  voice_channel_id?: string;
  text_channel_id?: string;
  playing: boolean;
  paused: boolean;
  position_ms: number;
  volume: number;
  loop: LoopMode;
  shuffle: boolean;
  current: Track | null;
  queue: Track[];
  queue_length: number;
  history: Track[];
  session_start: string;
  updated_at: string;
  can_control: boolean;
};

export type Guild = {
  id: string;
  name: string;
  icon: string | null;
  can_manage: boolean;
  bot_voice_channel_id?: string;
  user_voice_channel_id?: string;
  playing: boolean;
  current_title?: string;
  listening_with_bot: boolean;
};

export type Me = {
  id: string;
  username: string;
  global_name: string | null;
  avatar: string | null;
  admin: boolean;
  session_expires_at: string;
};

export type Playlist = {
  id: number;
  name: string;
  owner_id: string;
  created_at: string;
  track_count: number;
};

export type PlaylistTrack = {
  id: number;
  added_at: string;
  added_by: string;
  track: Track;
};

export type SearchResult = Track & { value: string };

export type Stats = {
  guilds: number;
  players: number;
  playing: number;
  uptime_seconds: number;
  radio_stations?: number;
  bot?: { id: string; username?: string; avatar_url?: string };
};

export type Song = {
  id: string;
  text: string;
  artist: string;
  title: string;
  album: string;
  art: string;
};

export type Station = {
  id: number;
  shortcode: string;
  name: string;
  description: string;
  online: boolean;
  listeners: number;
  public_player_url: string;
  stream_url: string;
  live: { is_live: boolean; streamer_name: string; art: string | null };
  now_playing: {
    played_at: number;
    duration: number;
    elapsed: number;
    remaining: number;
    playlist: string;
    is_request: boolean;
    song: Song;
  } | null;
  playing_next?: { song: Song };
  song_history?: { played_at: number; song: Song }[];
};

export type NodeInfo = {
  name: string;
  status: string;
  players: number;
  playing_players: number;
  uptime_ms: number;
  cpu_load: number;
  system_load: number;
  cores: number;
  memory_used: number;
  memory_allocated: number;
  frames_sent: number;
  frames_nulled: number;
  frames_deficit: number;
};

export type StatusComponent = {
  id: string;
  name: string;
  group: string;
  status: "operational" | "down";
  detail: string;
  uptime: number | null;
  history: { start: string; uptime: number | null }[];
};

export type SystemStatus = {
  status: "operational" | "degraded" | "outage";
  checked_at: string;
  tracking_since: string;
  bucket_minutes: number;
  components: StatusComponent[];
  nodes: NodeInfo[];
  bot: {
    guilds: number;
    players: number;
    playing: number;
    uptime_seconds: number;
    gateway_latency_ms?: number;
  };
};

export type LyricsResult = {
  track: Track;
  lyrics: {
    title: string;
    artist: string;
    album?: string;
    duration_ms: number;
    instrumental: boolean;
    plain: string;
    synced?: { time_ms: number; text: string }[];
    source: string;
  };
};
