import { Buffer } from "node:buffer";
import { typeByExtension } from "@std/media-types";
import { s3mini } from "s3mini";

export const uploadToS3 = async (
  filePath: string,
  fileName: string,
  s3Options: {
    accessKeyId: string;
    secretAccessKey: string;
    endpoint: string;
    region: string;
  },
) => {
  console.log(`Uploading backup to S3: ${fileName}`);

  try {
    const s3Client = new s3mini(s3Options);
    const file = await Deno.readFile(filePath);
    const fileBuffer = Buffer.from(file);

    await s3Client.putObject(fileName, fileBuffer, typeByExtension(fileName));

    console.log(`Backup uploaded to S3: ${fileName}`);
  } catch (error) {
    console.error(
      `Failed to upload file ${fileName} to S3:`,
      error instanceof Error ? error.message : String(error),
    );
    Deno.exit(1);
  }
};
