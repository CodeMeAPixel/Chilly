export type CommandGroup = {
  name: string;
  commands: { name: string; description: string }[];
};

export const commandGroups: CommandGroup[] = [
  {
    name: "Music",
    commands: [
      { name: "/play", description: "Play a song, album or playlist from a search or link" },
      { name: "/search", description: "Search with live suggestions and pick the exact track" },
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
      { name: "/list add", description: "Add tracks or whole playlists to a playlist" },
      { name: "/list remove", description: "Remove a track from a playlist" },
      { name: "/list delete", description: "Delete a playlist" },
      { name: "/list list", description: "List your playlists" },
    ],
  },
  {
    name: "Radio",
    commands: [
      { name: "/radio play", description: "Tune into one of our 24/7 radio stations" },
      { name: "/radio now", description: "See what's on air right now" },
      { name: "/radio stations", description: "Browse every station" },
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
