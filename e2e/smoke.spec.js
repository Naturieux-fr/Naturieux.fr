// Smoke E2E: the full stack (Go server + SQLite + SPA) boots, a player can
// register, and the quiz setup screen appears. Runs against a server started
// with DEV_MODE=1 (mock species) and REGISTRATION_MODE=open.
const { test, expect } = require('@playwright/test');

test('register and reach the quiz setup', async ({ page }) => {
  await page.goto('/');

  const name = 'E2E' + Math.floor(Math.random() * 1e6);
  await page.getByPlaceholder('Nom de naturaliste').first().fill(name);
  await page.getByPlaceholder(/Mot de passe/).first().fill('e2epass1');
  await page.getByRole('button', { name: /Créer mon carnet/ }).click();

  // Once registered, the member card and the "Préparer une sortie" plate show.
  await expect(page.getByText('Préparer une sortie')).toBeVisible();
  await expect(page.getByText(name)).toBeVisible();
});
