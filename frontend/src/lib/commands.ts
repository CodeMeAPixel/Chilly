export type CommandGroup = {
  name: string;
  commands: { name: string; description: string }[];
};

export const commandGroups: CommandGroup[] = [
  {
    name: "Radio",
    commands: [
      { name: "/radio play", description: "Tune your voice channel into one of our 24/7 stations" },
      { name: "/radio now", description: "See what's on air right now" },
      { name: "/radio stations", description: "Browse every station and who's listening" },
    ],
  },
  {
    name: "24/7",
    commands: [
      { name: "/247 on", description: "Keep a station playing in your channel around the clock, even when it's empty" },
      { name: "/247 off", description: "Turn 24/7 radio off again" },
      { name: "/247 status", description: "See which station is set to play 24/7 and where" },
    ],
  },
  {
    name: "Requests",
    commands: [
      { name: "/request", description: "Ask a station to play a song from the library soon" },
      { name: "/suggest", description: "Suggest a song that isn't in the library yet" },
      { name: "/requests", description: "See your requests and suggestions and where they're at" },
    ],
  },
  {
    name: "Music",
    commands: [
      { name: "/play", description: "Play a song, album, artist or playlist from the Chilly library" },
      { name: "/search", description: "Search the library with live suggestions and pick the exact song" },
      { name: "/queue", description: "See what's playing and what's up next" },
      { name: "/now", description: "Show the current track with a progress bar" },
      { name: "/lyrics", description: "Show lyrics for the current song or any song" },
      { name: "/skip", description: "Skip to the next track" },
      { name: "/pause", description: "Pause playback" },
      { name: "/resume", description: "Resume playback" },
      { name: "/seek", description: "Jump to a position in the current track" },
      { name: "/loop", description: "Loop the track or the whole queue" },
      { name: "/shuffle", description: "Toggle shuffle mode" },
      { name: "/remove", description: "Remove a track from the queue" },
      { name: "/stop", description: "Stop and clear the queue" },
    ],
  },
  {
    name: "Playlists",
    commands: [
      { name: "/playlist", description: "Play one of your saved playlists" },
      { name: "/list create", description: "Create a new playlist" },
      { name: "/list add", description: "Add songs, albums or artists from the library to a playlist" },
      { name: "/list remove", description: "Remove a track from a playlist" },
      { name: "/list delete", description: "Delete a playlist" },
      { name: "/list list", description: "List your playlists" },
    ],
  },
  {
    name: "General",
    commands: [
      { name: "/join", description: "Bring Chilly into your voice channel" },
      { name: "/leave", description: "Send Chilly home" },
      { name: "/help", description: "List every command" },
      { name: "/invite", description: "Get an invite link" },
    ],
  },
];
