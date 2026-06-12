# 編碼規範

本文整理 `buy-ticket` 專案在 Windows / PowerShell 環境下，修改中文檔案時的編碼注意事項。

---

## 1. 統一規則

- 所有文字檔一律使用 `UTF-8`
- 中文文件不要用 PowerShell 管線直接整份覆蓋
- 優先使用 `apply_patch` 修改程式碼與文件

---

## 2. 為什麼會亂碼

常見原因有三個：

- PowerShell 主控台輸出編碼不是 UTF-8
- `Set-Content` / `Out-File` 重寫時改掉原本編碼
- 原始檔案本來就不是乾淨 UTF-8

---

## 3. 建議做法

### 最優先

修改以下檔案時，優先用 `apply_patch`：

- `.go`
- `.md`
- `.yaml`
- `.json`
- `.sql`

這樣最不容易把中文寫壞。

### 不建議

避免這類寫法：

```powershell
Get-Content file.md | Set-Content file.md
```

或：

```powershell
Get-Content file.md | Out-File file.md
```

這種最容易把中文檔編碼改壞。

---

## 4. 如果一定要用 PowerShell 寫檔

請用 `.NET` 明確指定 UTF-8 No BOM：

```powershell
$path = "D:\SourceCode\Go\buy-ticket\doc\api-flow.md"
$content = Get-Content $path -Raw
$utf8NoBom = New-Object System.Text.UTF8Encoding($false)
[System.IO.File]::WriteAllText($path, $content, $utf8NoBom)
```

重點：

- `Get-Content` 要加 `-Raw`
- `UTF8Encoding($false)` 表示 UTF-8 No BOM
- 不要再接 `Set-Content`

---

## 5. PowerShell 顯示設定

這只能改善終端顯示，不保證寫檔安全：

```powershell
[Console]::InputEncoding = [System.Text.UTF8Encoding]::new($false)
[Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
chcp 65001
```

如果終端看到亂碼，不代表檔案本身一定壞掉。

---

## 6. 檢查方式

如果懷疑終端顯示有問題，請直接用編輯器確認：

- 用 VS Code 開檔
- 確認右下角編碼是 `UTF-8`
- 以編輯器顯示結果為準，不要只看 PowerShell 輸出

---

## 7. 專案建議

這個專案後續建議：

- 中文文件：用 `apply_patch`
- 程式碼：用 `apply_patch`
- 只有在必要時才用 PowerShell 重寫整份文字檔

這樣最穩。
