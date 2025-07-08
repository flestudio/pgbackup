import { $ } from "@david/dax";
import { format } from "@std/datetime";
import { join } from "@std/path";
import { compress } from "./compress.ts";
import { createBackup } from "./create-backup.ts";
import { parseEnv } from "./parse-env.ts";
import { generateDiscordPayload, sendDiscordWebhook } from "./send-webhook.ts";
import { uploadToS3 } from "./upload-s3.ts";

const TMP_DIRECTORY = "/tmp/backups/";
const CURRENT_DIRECTORY = await $`pwd`.stdout("piped").text();
const RAW_FILENAME = `backup_${format(new Date(), "yyyyMMdd_HHmmss")}.sql`;
const COMPRESSED_FILENAME = `${RAW_FILENAME}.zst`;
const COMPRESSED_FILEPATH = join(TMP_DIRECTORY, COMPRESSED_FILENAME);

const main = async () => {
  const { database, out, s3Options, discordWebhookUrl } = parseEnv(
    Deno.env.toObject(),
  );

  try {
    // pg_dump
    const { info, path } = await createBackup(database, COMPRESSED_FILEPATH);

    // zstd
    const backupFilePath = join(CURRENT_DIRECTORY, out, COMPRESSED_FILENAME);
    await compress(COMPRESSED_FILEPATH, backupFilePath);

    // rm
    await $`rm ${COMPRESSED_FILEPATH}`;
    console.log(`Temporary file ${COMPRESSED_FILEPATH} removed`);

    // Upload to S3
    const fileName = join("backups", COMPRESSED_FILENAME);
    await uploadToS3(backupFilePath, fileName, s3Options);

    // Send Discord webhook
    const payload = await generateDiscordPayload(info, path);
    await sendDiscordWebhook(discordWebhookUrl, payload);

    // Remove old backups
    await $`find ${join(CURRENT_DIRECTORY, out)} -name 'backup_*.sql.zst' -daystart -mtime +7 -delete`;
  } catch (error) {
    if (!(error instanceof Error)) return;

    console.error(error.message);

    await sendDiscordWebhook(discordWebhookUrl, {
      content: "🔥 バックアップの作成に失敗しました",
      embeds: [
        {
          title: "エラー",
          description: error.message,
        },
      ],
    });

    Deno.exit(1);
  }
};

main();
