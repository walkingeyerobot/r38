import { Request, Response } from 'express';
import fs from 'fs';

export function serveIndex(req: Request, res: Response) {
  fs.readFile(`dev_server/srv/index.html`, 'utf8', (err, data) => {
    if (err) {
      res.status(500).send(err.message);
    } else {
      const payload = {
        auth: {
          id: 47,
          name: 'Lilith',
        },
      };
      const transformed =
          data.replace(
              `{{authInfo}}`,
              `const PAYLOAD = ${JSON.stringify(payload)}`);
      res.send(transformed);
    }
  });
}
