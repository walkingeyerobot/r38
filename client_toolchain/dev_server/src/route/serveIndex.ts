import { Request, Response } from 'express';
import fs from 'fs';

export function serveIndex(req: Request, res: Response) {
  fs.readFile(`dev_server/srv/index.html`, 'utf8', (err, data) => {
    if (err) {
      res.status(500).send(err.message);
    } else {
      let replacementStr: string;
      if (req.query['noAuth'] == '' || req.query['noAuth'] == 'true') {
        replacementStr = '';
      } else {
        const userInfo = {
          name: 'red_weather',
          picture: `https://cdn.discordapp.com/avatars/117108584017428481/f91aadd54de1929aaad167cabc99bdb1.png`,
          userId: 5,
        };

        replacementStr =
            `window.UserInfo=${JSON.stringify(JSON.stringify(userInfo))};`;
      }

      const transformed = data.replace(`{{authInfo}}`, replacementStr);
      res.send(transformed);
    }
  });
}
