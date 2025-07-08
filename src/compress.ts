import { $ } from "@david/dax";

export const compress = async (fromPath: string, toPath: string) => {
  console.log(`Compressing file from ${fromPath} to ${toPath}`);

  try {
    await $`zstd -8 -T0 ${fromPath} -o ${toPath}`;
    console.log(`File compressed successfully to ${toPath}`);
  } catch (error) {
    console.error(
      `Failed to compress file ${fromPath}:`,
      error instanceof Error ? error.message : String(error),
    );

    await Deno.remove(fromPath).catch(() => {
      console.error(`Failed to remove temporary file ${fromPath}`);
    });

    Deno.exit(1);
  }
};
