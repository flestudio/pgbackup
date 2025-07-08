import $ from "@david/dax";

export const getDiskInfo = async () => {
  const diskInfo = await $`df -h /`.stdout("piped").text();

  const lines = diskInfo.trim().split("\n");
  const values = lines[1].split(/\s+/);
  const diskUsage = {
    filesystem: values[0],
    size: values[1],
    used: values[2],
    available: values[3],
    usePercentage: values[4],
    mountedOn: values[5],
  };

  return diskUsage;
};
