import * as fs from "fs";
import * as os from "os";
import * as path from "path";

export const endpoint = "https://api.hdr.is";

interface Config {
  hdrApiKey?: string;
}

export const loadConfigFile = (): Config => {
  try {
    const tudorConfigPath = path.resolve(os.homedir(), ".tudorrc");
    return JSON.parse(fs.readFileSync(tudorConfigPath, "utf8"));
  } catch (error) {
    return {};
  }
};
