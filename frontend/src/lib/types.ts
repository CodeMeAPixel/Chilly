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
  can_manage: boolean;
  stay: Stay | null;
};

export type Stay = {
  guild_id: string;
  voice_channel_id: string;
  text_channel_id?: string;
  station: string;
  station_name: string;
  enabled_by: string;
  enabled_at: string;
  health: { failures: number; last_error?: string; retry_at?: string };
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
  available: boolean;
  album?: string;
};

export type LibraryPlaylist = {
  name: string;
  track_count: number;
  art?: string;
  artists: string[];
};

export type SearchResult = Track & { value: string; album?: string; station?: string; playlists?: string[] };

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
  location?: string;
  status: string;
  players: number;
  playing_players: number;
  uptime_ms: number;
  cpu_load: number;
  system_load: number;
  cores: number;
  memory_used: number;
  memory_allocated: number;
  memory_reservable: number;
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

export type AdminOverview = {
  version: string;
  go_version: string;
  started_at: string;
  uptime_seconds: number;
  goroutines: number;
  heap_bytes: number;
  sys_bytes: number;
  gateway_status: string;
  gateway_latency_ms: number;
  guilds: number;
  members: number;
  players: number;
  playing: number;
  radio: { enabled: boolean; healthy: boolean; last_poll?: string; stations: number; online: number };
  library: LibraryStatus;
  lyrics_backfill: {
    enabled: boolean;
    last_run?: string;
    written: number;
    not_found: number;
    pending: number;
    last_error?: string;
  };
  stays: Stay[];
  nodes: NodeInfo[];
};

export type AdminGuild = {
  id: string;
  name: string;
  icon_url: string | null;
  member_count: number;
  owner_id: string;
  joined_at: string;
  voice_channel_id?: string;
  listeners: number;
  playing: boolean;
  paused: boolean;
  current?: string;
  source?: string;
  radio?: string;
  queue_length: number;
  node?: string;
  stay: Stay | null;
};

export type LibraryStatus = {
  enabled: boolean;
  tracks: number;
  last_sync?: string;
  error?: string;
  media_url?: string;
};

export type AdminSearchResult = {
  total: number;
  tracks: SearchResult[];
  library: LibraryStatus;
  playback_test?: {
    node: string;
    track: string;
    took_ms: number;
    ok: boolean;
    length_ms?: number;
    error?: string;
    cause?: string;
  };
};

export type LogEntry = {
  time: string;
  level: "DEBUG" | "INFO" | "WARN" | "ERROR" | string;
  message: string;
  attrs?: Record<string, string>;
};

export type RequestStatus = "queued" | "playing" | "played" | "expired";
export type SuggestionStatus = "pending" | "reviewing" | "added" | "declined";

export type SongRequest = {
  id: number;
  user_id: string;
  guild_id?: string;
  station: string;
  media_key: string;
  title: string;
  artist: string;
  art?: string;
  source: "web" | "discord";
  status: RequestStatus;
  created_at: string;
  played_at?: string;
};

export type SongSuggestion = {
  id: number;
  user_id: string;
  artist: string;
  title: string;
  link?: string;
  note?: string;
  status: SuggestionStatus;
  reason?: string;
  library_key?: string;
  reviewed_by?: string;
  created_at: string;
  updated_at: string;
};

export type LibraryGroupSummary = {
  name: string;
  artist?: string;
  track_count: number;
  art?: string;
};

export type LibrarySummary = {
  tracks: number;
  artists: number;
  albums: number;
  playlists: number;
  synced_at?: string;
  requests?: { cooldown_seconds: number; max_pending: number; max_open_suggestions: number };
};

export type LibraryPage = {
  tracks: SearchResult[];
  total: number;
  page: number;
  per_page: number;
};
