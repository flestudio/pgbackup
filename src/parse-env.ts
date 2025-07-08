export const parseEnv = (env: Record<string, string>) => {
  const {
    DATABASE = "misskey",
    OUTPUT_DIR = "./backups",
    S3_BUCKET,
    S3_REGION = "auto",
    S3_ACCESS_KEY,
    S3_SECRET_KEY,
    S3_ENDPOINT,
    DISCORD_WEBHOOK_URL,
  } = env;

  if (S3_BUCKET === undefined || S3_BUCKET === "") {
    console.warn("S3_BUCKET is not set.");
    Deno.exit(1);
  }

  if (S3_ACCESS_KEY === undefined || S3_ACCESS_KEY === "") {
    console.warn("S3_ACCESS_KEY is not set.");
    Deno.exit(1);
  }

  if (S3_SECRET_KEY === undefined || S3_SECRET_KEY === "") {
    console.warn("S3_SECRET_KEY is not set.");
    Deno.exit(1);
  }

  if (S3_ENDPOINT === undefined || S3_ENDPOINT === "") {
    console.warn("S3_ENDPOINT is not set.");
    Deno.exit(1);
  }

  if (DISCORD_WEBHOOK_URL === undefined || DISCORD_WEBHOOK_URL === "") {
    console.warn("DISCORD_WEBHOOK_URL is not set.");
    Deno.exit(1);
  }

  const s3Endpoint = new URL(S3_BUCKET, S3_ENDPOINT).toString();

  return {
    database: DATABASE,
    out: OUTPUT_DIR,
    s3Options: {
      accessKeyId: S3_ACCESS_KEY,
      secretAccessKey: S3_SECRET_KEY,
      endpoint: s3Endpoint,
      region: S3_REGION,
    },
    discordWebhookUrl: DISCORD_WEBHOOK_URL,
  };
};
