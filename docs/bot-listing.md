# Bot listing kit

Copy for listing Chilly on top.gg, discordbotlist.com, discords.com, Discord's App Directory and similar sites. Keep the wording consistent with the [brand page](https://chillybot.space/brand), which also has the logo, banner and colours.

Replace `{CLIENT_ID}` with the bot's application ID before you paste anything.

## Basics

| Field | Value |
| --- | --- |
| Name | Chilly |
| Prefix | `/` (slash commands only) |
| Website | https://chillybot.space |
| Stations | https://chillybot.space/radio |
| Status | https://chillybot.space/status |
| Source code | https://github.com/CodeMeAPixel/Chilly |
| Invite | `https://discord.com/oauth2/authorize?client_id={CLIENT_ID}&scope=bot%20applications.commands&permissions=37047296` |
| Avatar | https://chillybot.space/brand/logo.png (512×512) |
| Banner | https://chillybot.space/opengraph-image (1200×630) |

## Tagline

> 24/7 radio for your Discord server.

## Short description

Most sites cap this at 140 to 200 characters. This version is 130:

> 24/7 radio for your Discord server. Tune into always-on stations, keep them playing around the clock and play any song on request.

Shorter alternative (101 characters):

> Always-on radio stations for your voice channels, with 24/7 mode, a live dashboard and song requests.

## Tags

Pick the ones the site offers, in this order of priority:

`Radio`, `24/7`, `Music`, `Lo-fi`, `Chill`, `Hip-Hop`, `Dashboard`, `Lyrics`, `Playlists`, `Free`

## Long description

Paste this as Markdown. Sites that don't render Markdown show it as plain text, which still reads fine.

```markdown
# 📻 Chilly — 24/7 radio for your Discord server

Chilly turns any voice channel into a radio station. Pick one of our always-on stations, press play and let it run. Turn on 24/7 mode and Chilly stays in the channel around the clock, even when nobody is listening, and comes back by itself after a restart.

## Why Chilly?

- **Our own stations.** Run by the Chilly team and streaming around the clock, with live now-playing info right in your channel.
- **True 24/7 mode.** `/247 on` keeps a station playing in your channel forever. No more rejoining the bot every morning.
- **Listen anywhere.** Every station also plays in your browser at chillybot.space/radio.
- **A live dashboard.** Switch stations, skip, seek and manage the queue from your browser. Changes show up instantly for everyone.
- **Your songs, too.** Want something specific? `/play` finds it on YouTube, SoundCloud or Spotify, with lyrics and saved playlists.
- **Built to stay up.** Redundant audio servers with automatic failover, and a public status page.
- **Free and open source.** No premium tier, no vote locks, no ads.

## Get started

1. Invite Chilly and join a voice channel.
2. Run `/radio play` and pick a station.
3. Want it on all the time? Run `/247 on`. You'll need the Manage Server permission.

## Commands

**Radio**
- `/radio play` — tune your voice channel into a station
- `/radio now` — see what's on air
- `/radio stations` — browse every station

**24/7**
- `/247 on` — keep a station playing around the clock
- `/247 off` — turn 24/7 mode off
- `/247 status` — see the current 24/7 setting

**Music**
- `/play`, `/search` — play a song, album or playlist
- `/queue`, `/now`, `/lyrics` — see what's playing
- `/pause`, `/resume`, `/skip`, `/seek`, `/stop`, `/shuffle`, `/loop`, `/remove` — control playback
- `/playlist`, `/list` — save and play your own playlists

**General**
- `/join`, `/leave`, `/help`, `/invite`

## Links

🌐 Website: https://chillybot.space
📻 Stations: https://chillybot.space/radio
🟢 Status: https://chillybot.space/status
💻 Source: https://github.com/CodeMeAPixel/Chilly
```

## Permissions

The invite link asks only for what Chilly uses. Bot lists often ask you to justify each one:

| Permission | Why |
| --- | --- |
| View Channel, Read Message History | See the channel it was used in |
| Send Messages, Embed Links, Use External Emojis | Post the now-playing card and command replies |
| Connect, Speak, Use Voice Activity | Join the voice channel and play audio |

Chilly never needs Administrator, Manage Server or any moderation permission. The `/247` command is limited to members with Manage Server by default; server owners can change that under Server Settings → Integrations.

## FAQ snippets

**Is Chilly free?** Yes. Every feature is free and there's no premium tier.

**Does it really stay in the channel forever?** With `/247 on`, yes. Chilly rejoins after restarts or disconnects and keeps the station playing until someone runs `/247 off`.

**Can I play my own music?** Yes. `/play` takes a song name or a YouTube, SoundCloud or Spotify link.

**Where can I see if something is down?** https://chillybot.space/status shows live health for every part of the bot.
