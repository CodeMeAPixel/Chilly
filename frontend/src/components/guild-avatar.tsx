import { cn, guildIcon, initials } from "@/lib/format";
import type { Guild } from "@/lib/types";

export function GuildAvatar({ guild, size = "md" }: { guild: Pick<Guild, "id" | "name" | "icon">; size?: "sm" | "md" | "lg" }) {
  const sizes = { sm: "h-6 w-6 rounded-lg text-[10px]", md: "h-12 w-12 rounded-2xl text-sm", lg: "h-16 w-16 rounded-2xl text-base" };
  const icon = guildIcon(guild.id, guild.icon);
  if (icon) {
    return <img src={icon} alt="" className={cn("shrink-0 object-cover", sizes[size])} />;
  }
  return (
    <div className={cn("grid shrink-0 place-items-center bg-surface-2 font-semibold text-muted", sizes[size])}>
      {initials(guild.name)}
    </div>
  );
}
