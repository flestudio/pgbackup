import { $ } from "@david/dax";

export const compress = async (fromPath: string, toPath: string) => {
  console.log(`Compressing file from ${fromPath} to ${toPath}`);

  try {
    await $`zstd -8 -T0 ${fromPath} -o ${toPath}`;
    console.log(`File compressed successfully to ${toPath}`);

    const stats = await Deno.stat(toPath);
    return { info: stats, path: toPath };
  } catch (error) {
    await Deno.remove(fromPath).catch(() => {
      console.error(`Failed to remove temporary file ${fromPath}`);
    });

    throw new Error(
      `Failed to compress file ${fromPath}: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
  }
};
