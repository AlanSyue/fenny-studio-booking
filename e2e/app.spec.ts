import { test, expect } from '@playwright/test'

test.describe('前台問卷頁面', () => {
  test('應顯示表單標題和所有欄位', async ({ page }) => {
    await page.goto('/')
    await expect(page.locator('h1')).toHaveText('Fenny Studio')
    await expect(page.locator('text=手作課程詢問表單')).toBeVisible()
    await expect(page.locator('text=怎麼稱呼')).toBeVisible()
    await expect(page.locator('text=Email')).toBeVisible()
    await expect(page.locator('text=想要詢問的手作課程項目')).toBeVisible()
    await expect(page.locator('text=預估人數')).toBeVisible()
    await expect(page.locator('text=其他聯絡方式補充')).toBeVisible()
    await expect(page.locator('button[type="submit"]')).toHaveText('送出詢問')
  })

  test('送出表單成功後顯示感謝畫面', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=感謝您的詢問')).toBeVisible({ timeout: 5000 })
    await expect(page.locator('text=填寫新的詢問')).toBeVisible()
  })

  test('點擊「填寫新的詢問」可重新填寫', async ({ page }) => {
    await page.goto('/')
    await page.fill('input[type="email"]', 'test@example.com')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=感謝您的詢問')).toBeVisible({ timeout: 5000 })
    await page.click('text=填寫新的詢問')
    await expect(page.locator('button[type="submit"]')).toHaveText('送出詢問')
  })
})

test.describe('後台登入', () => {
  test('應顯示登入表單', async ({ page }) => {
    await page.goto('/admin/login')
    await expect(page.locator('text=後台管理登入')).toBeVisible()
    await expect(page.locator('input[type="text"]')).toBeVisible()
    await expect(page.locator('input[type="password"]')).toBeVisible()
  })

  test('錯誤帳密應顯示錯誤訊息', async ({ page }) => {
    await page.goto('/admin/login')
    await page.fill('input[type="text"]', 'wrong')
    await page.fill('input[type="password"]', 'wrong')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=帳號或密碼錯誤')).toBeVisible({ timeout: 5000 })
  })

  test('正確帳密應導向課程管理頁', async ({ page }) => {
    await page.goto('/admin/login')
    await page.fill('input[type="text"]', 'admin')
    await page.fill('input[type="password"]', 'admin123')
    await page.click('button[type="submit"]')
    await page.waitForURL('**/admin/courses', { timeout: 5000 })
    await expect(page.locator('h1:has-text("課程管理")')).toBeVisible()
  })

  test('未登入應導向登入頁', async ({ page }) => {
    await page.goto('/admin/courses')
    await page.waitForURL('**/admin/login')
  })
})

test.describe('後台課程管理', () => {
  test.beforeEach(async ({ page }) => {
    // Login first
    await page.goto('/admin/login')
    await page.fill('input[type="text"]', 'admin')
    await page.fill('input[type="password"]', 'admin123')
    await page.click('button[type="submit"]')
    await page.waitForURL('**/admin/courses', { timeout: 5000 })
  })

  test('可新增課程', async ({ page }) => {
    await page.click('text=新增課程')
    await page.fill('input[placeholder="輸入課程名稱"]', '手工皂課程')
    await page.click('button:has-text("儲存")')
    await expect(page.locator('text=手工皂課程').first()).toBeVisible({ timeout: 5000 })
  })

  test('可編輯課程', async ({ page }) => {
    // Ensure there's a course first
    await page.click('text=新增課程')
    await page.fill('input[placeholder="輸入課程名稱"]', '編輯測試課程')
    await page.click('button:has-text("儲存")')
    await expect(page.locator('h3:has-text("編輯測試課程")').first()).toBeVisible({ timeout: 5000 })

    // Edit it - find the card's edit button
    const card = page.locator('.rounded-xl', { has: page.locator('h3:has-text("編輯測試課程")') }).first()
    await card.locator('button:has-text("編輯"), a:has-text("編輯")').first().click()
    // Wait for form to appear and clear + fill
    const nameInput = page.locator('input[placeholder="輸入課程名稱"]')
    await nameInput.waitFor({ state: 'visible', timeout: 5000 })
    await nameInput.clear()
    await nameInput.fill('已修改課程')
    await page.click('button:has-text("儲存")')
    await expect(page.locator('h3:has-text("已修改課程")').first()).toBeVisible({ timeout: 5000 })
  })

  test('導覽列可切換到詢問列表', async ({ page }) => {
    await page.click('nav >> text=詢問列表')
    await page.waitForURL('**/admin/inquiries')
    await expect(page.locator('h1:has-text("詢問列表")')).toBeVisible()
  })
})

test.describe('後台詢問列表', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/admin/login')
    await page.fill('input[type="text"]', 'admin')
    await page.fill('input[type="password"]', 'admin123')
    await page.click('button[type="submit"]')
    await page.waitForURL('**/admin/courses', { timeout: 5000 })
    await page.click('nav >> text=詢問列表')
    await page.waitForURL('**/admin/inquiries')
  })

  test('應顯示篩選按鈕', async ({ page }) => {
    await expect(page.locator('button:has-text("全部")').first()).toBeVisible()
    await expect(page.locator('button:has-text("未處理")').first()).toBeVisible()
    await expect(page.locator('button:has-text("已處理")').first()).toBeVisible()
  })

  test('應顯示先前提交的詢問', async ({ page }) => {
    // The inquiry from the form test should appear
    await expect(page.locator('text=test@example.com').first()).toBeVisible({ timeout: 5000 })
  })

  test('可切換處理狀態', async ({ page }) => {
    const statusBtn = page.locator('button:has-text("未處理")').last()
    await statusBtn.click()
    await expect(page.locator('button:has-text("已處理")').last()).toBeVisible({ timeout: 5000 })
  })
})

test.describe('端對端流程', () => {
  test('前台送出 → 後台可見並標記處理', async ({ page }) => {
    // Step 1: Submit inquiry from public form
    await page.goto('/')
    await page.fill('input[placeholder="您的稱呼"]', 'E2E 測試者')
    await page.fill('input[type="email"]', 'e2e@test.com')
    await page.fill('input[placeholder="預估參加人數"]', '5')
    await page.fill('textarea', 'LINE: test123')
    await page.click('button[type="submit"]')
    await expect(page.locator('text=感謝您的詢問')).toBeVisible({ timeout: 5000 })

    // Step 2: Login to admin
    await page.goto('/admin/login')
    await page.fill('input[type="text"]', 'admin')
    await page.fill('input[type="password"]', 'admin123')
    await page.click('button[type="submit"]')
    await page.waitForURL('**/admin/courses', { timeout: 5000 })

    // Step 3: Navigate to inquiry list
    await page.click('nav >> text=詢問列表')
    await page.waitForURL('**/admin/inquiries')

    // Step 4: Verify the inquiry is visible
    await expect(page.locator('text=e2e@test.com').first()).toBeVisible({ timeout: 5000 })
    await expect(page.locator('text=E2E 測試者').first()).toBeVisible()

    // Step 5: Mark as processed
    const row = page.locator('tr', { has: page.locator('text=e2e@test.com') }).first()
    await row.locator('button:has-text("未處理")').click()
    await expect(row.locator('button:has-text("已處理")')).toBeVisible({ timeout: 5000 })
  })
})
