#!/usr/bin/env node
"use strict";

const got = require('got');

console.log('Calling Travis...');

got.post(`https://api.travis-ci.org/repo/rackn%2Frackn-saas/requests`, {
  headers: {
    "Content-Type": "application/json",
    "Accept": "application/json",
    "Travis-API-Version": "3",
    "Authorization": `token ${process.env.TRAVIS_API_TOKEN}`,
  },
  body: JSON.stringify({
    request: {
      message: `Trigger build at digitalrebar/provision`,
      branch: 'tip',
    },
  }),
})
.then(() => {
  console.log("Triggered build on behalf of digitalrebar/provision at rackn/rackn-saas");
})
.catch((err) => {
  console.error(err);
  process.exit(-1);
});
