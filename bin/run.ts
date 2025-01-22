import yargs, { CommandModule } from "yargs";
import { config } from "dotenv";
import { commands } from "../src";
import pc from "picocolors";

config();

const run = yargs(process.argv.slice(2)).scriptName("vers").usage(pc.magenta("Usage: $0 <command> [options]"));
for (const command of commands) {
  run.command(command as CommandModule);
}

run.demandCommand(1, "You need at least one command before moving on").help().argv;
