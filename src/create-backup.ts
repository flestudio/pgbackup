import { $ } from "@david/dax";

export const createBackup = async (databaseName: string, backupTo: string) => {
  console.log(`Creating backup for database: ${databaseName}`);

  try {
    await $`pg_dump -U postgres -d ${databaseName} -f ${backupTo}`;
    console.log(`Backup created at ${backupTo}`);
  } catch (error) {
    throw new Error(
      `Failed to create backup for database ${databaseName}: ${
        error instanceof Error ? error.message : String(error)
      }`,
    );
  }
};
