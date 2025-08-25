import 'dotenv/config'
import fs from 'fs'
import csv from 'csv-parser'
import crypto from 'crypto'
import sql from 'mssql'

const CATEGORY_RULES = [
  { category: 'Coffee',          match: ['starbucks', 'dunkin'] },
  { category: 'Transportation',  match: ['uber', 'lyft'] },
  { category: 'Groceries',       match: ['whole foods', 'trader joes', 'kroger', 'aldi'] },
  { category: 'Dining',          match: ['mcdonalds', 'chipotle', 'dominos', 'ubereats', 'doordash'] },
  { category: 'Fuel',            match: ['shell', 'chevron', 'bp'] },
  { category: 'Entertainment',   match: ['netflix', 'spotify', 'hulu'] },
  { category: 'Utilities',       match: ['verizon', 'comcast', 'att'] },
];

const dbConfig = {
  server: process.env.Server,
  user: process.env.User,
  password: process.env.Password,
  database: process.env.Database,
  port: parseInt(process.env.Port || '1433', 10),
  options: { encrypt: true, trustServerCertificate: true },
}

function parseCsv(path) {
  return new Promise((resolve, reject) => {
    const rows = []
    fs.createReadStream(path)
      .pipe(csv())
      .on('data', (row) => rows.push(row))
      .on('end', () => resolve(rows))
      .on('error', reject)
  })
}

function canonicalImportKey({ userId, date, amount, merchant, category, description }) {
  const d = String(date).trim()                 
  const a = Number(amount)                     
  const m = String(merchant ?? '').trim().toLowerCase()
  const c = String(category ?? '').trim().toLowerCase()
  const desc = String(description ?? '').trim()
  return `${userId}|${d}|${a}|${m}|${c}|${desc}`
}

function sha256Hex(s) {
  return crypto.createHash('sha256').update(s, 'utf8').digest('hex')
}

async function resolveUserId(pool) {
  if (process.env.USER_ID) return Number(process.env.USER_ID)
  const r = await pool.request().query('SELECT TOP (1) id FROM dbo.users ORDER BY id')
  if (r.recordset.length) return r.recordset[0].id
  throw new Error('No users found; create a user first (register endpoint).')
}

async function main() {
  const csvPath = process.argv[2] || './sample.csv'
  const rows = await parseCsv(csvPath)
  console.log(`Parsed ${rows.length} rows from ${csvPath}`)

  const pool = await sql.connect(dbConfig)
  try {
    const userId = await resolveUserId(pool)
    console.log(`Using user_id=${userId}`)

    const ps = new sql.PreparedStatement(pool)
    ps.input('user_id', sql.Int)
    ps.input('date', sql.Date)
    ps.input('amount', sql.Decimal(19, 4))
    ps.input('merchant', sql.NVarChar(100))
    ps.input('category', sql.NVarChar(80))
    ps.input('description', sql.NVarChar(1000))
    ps.input('import_id', sql.VarChar(128))

    await ps.prepare(`
      INSERT INTO dbo.transactions
        (user_id, [date], amount, merchant, category, description, import_id)
      VALUES
        (@user_id, @date, @amount, @merchant, @category, @description, @import_id)
    `)

    let inserted = 0, skipped = 0, failed = 0

    for (const raw of rows) {
      const date = String(raw.date).trim()                
      const amount = Number(raw.amount)                    
      const merchant = String(raw.merchant ?? '').trim()
      let category = String(raw.category ?? '').trim()
      const description = String(raw.description ?? '').trim()

    if (!category || category.toLowerCase() === 'uncategorized') {
        const mLower = merchant.toLowerCase()
        const rule = CATEGORY_RULES.find(r => r.match.some(k => mLower.includes(k)))
        category = rule ? rule.category : 'Uncategorized'
    }

      const key = canonicalImportKey({ userId, date, amount, merchant, category, description })
      const import_id = sha256Hex(key)

      try {
        await ps.execute({
          user_id: userId,
          date,
          amount,
          merchant,
          category,
          description,
          import_id,
        })
        inserted++
      } catch (err) {
        if (err && (err.number === 2627 || err.number === 2601)) {
          skipped++
        } else {
          failed++
          console.error(`Row failed:`, err?.message || err)
        }
      }
    }

    await ps.unprepare()
    console.log(`Done. Inserted: ${inserted}, Skipped (dupes): ${skipped}, Failed: ${failed}`)
  } finally {
    await sql.close()
  }
}

await main().catch((e) => {
  console.error(e)
  process.exit(1)
})
