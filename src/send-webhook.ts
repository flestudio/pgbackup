import { getDiskInfo } from "./disk-info.ts";
import { formatBytes } from "./utils.ts";

type DiscordEmbed = {
  title?: string;
  description?: string;
  url?: string;
  color?: number;
  timestamp?: string;
  footer?: {
    text: string;
    icon_url?: string;
  };
  image?: {
    url: string;
  };
  thumbnail?: {
    url: string;
  };
  author?: {
    name: string;
    url?: string;
    icon_url?: string;
  };
  fields?: {
    name: string;
    value: string;
    inline?: boolean;
  }[];
};

type DiscordWebhookPayload = {
  content?: string;
  embeds?: DiscordEmbed[];
  username?: string;
  avatar_url?: string;
  tts?: boolean;
};

export const sendDiscordWebhook = async (
  webhookUrl: string,
  payload: DiscordWebhookPayload,
): Promise<Response> => {
  const response = await fetch(webhookUrl, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: JSON.stringify(payload),
  });

  if (!response.ok) {
    throw new Error(`Failed to send Discord webhook: ${response.statusText}`);
  }

  return response;
};

export const generateDiscordPayload = async (
  backupFileInfo: Deno.FileInfo,
  backupFilePath: string,
) => {
  const diskUsage = await getDiskInfo();

  return {
    content: "✅ バックアップを作成しました",
    avatar_url: "https://github.com/flestudio.png",
    embeds: [
      {
        title: "バックアップ情報",
        fields: [
          {
            name: "ファイル",
            value: backupFilePath,
          },
          {
            name: "サイズ",
            value: formatBytes(backupFileInfo.size),
          },
          {
            name: "作成日時",
            value: new Date(`${backupFileInfo.birthtime}`).toISOString(),
          },
        ],
      },
      {
        title: "ディスク情報",
        fields: [
          {
            name: "総容量",
            value: diskUsage.size,
            inline: true,
          },
          {
            name: "使用中",
            value: diskUsage.used,
            inline: true,
          },
          {
            name: "空き",
            value: diskUsage.available,
            inline: true,
          },
          {
            name: "使用率",
            value: diskUsage.usePercentage,
            inline: true,
          },
        ],
      },
    ],
  } satisfies DiscordWebhookPayload;
};
