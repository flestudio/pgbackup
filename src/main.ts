import { cli, define } from "@kazupon/gunshi";

const command = define({
  name: "greeter",
  description: "A simple greeting CLI",
  args: {
    name: {
      type: "string",
      short: "n",
      description: "Name to greet",
    },
    uppercase: {
      type: "boolean",
      short: "u",
      description: "Convert greeting to uppercase",
    },
  },
  run: (ctx) => {
    const { name = "World", uppercase } = ctx.values;
    let greeting = `Hello, ${name}!`;

    if (uppercase) {
      greeting = greeting.toUpperCase();
    }

    console.log(greeting);
  },
});

await cli(Deno.args, command);
