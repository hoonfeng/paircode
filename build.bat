    copy /Y "companion.exe" "%DIST_DIR%\" >nul
if exist "bin\tesseract\tesseract.exe" copy "bin\tesseract\tesseract.exe" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
if exist "bin\tesseract\tesseract-uninstall.exe" copy "bin\tesseract\tesseract-uninstall.exe" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
if exist "bin\tesseract\winpath.exe" copy "bin\tesseract\winpath.exe" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
if exist "bin\tesseract\*.dll" copy "bin\tesseract\*.dll" "%DIST_DIR%\bin\tesseract\" >nul 2>&1
if exist "bin\tesseract\tessdata\*" xcopy /E /I /Y "bin\tesseract\tessdata\*" "%DIST_DIR%\bin\tesseract\tessdata\" >nul 2>&1
if exist "bin\config\models.json" copy "bin\config\models.json" "%DIST_DIR%\bin\config\" >nul 2>&1
if exist "bin\headless-check.js" copy "bin\headless-check.js" "%DIST_DIR%\bin\" >nul 2>&1
if exist "config\models.json" copy "config\models.json" "%DIST_DIR%\config\" >nul 2>&1
if exist "config\settings.template.json" copy "config\settings.template.json" "%DIST_DIR%\config\settings.json" >nul 2>&1
if exist "config\mcp.json" copy "config\mcp.json" "%DIST_DIR%\config\" >nul 2>&1
if exist "config\skills\*" xcopy /E /I /Y "config\skills\*" "%DIST_DIR%\config\skills\" >nul 2>&1
if exist "config\roles\*" xcopy /E /I /Y "config\roles\*" "%DIST_DIR%\config\roles\" >nul 2>&1
if exist "config\philosophy\*" xcopy /E /I /Y "config\philosophy\*" "%DIST_DIR%\config\philosophy\" >nul 2>&1
if exist "assets\icon.svg" copy "assets\icon.svg" "%DIST_DIR%\assets\" >nul 2>&1
if exist "assets\icon.png" copy "assets\icon.png" "%DIST_DIR%\assets\" >nul 2>&1
if exist "assets\icon64.png" copy "assets\icon64.png" "%DIST_DIR%\assets\" >nul 2>&1
if exist "assets\icon128.png" copy "assets\icon128.png" "%DIST_DIR%\assets\" >nul 2>&1
if exist "fonts\*.ttf" copy "fonts\*.ttf" "%DIST_DIR%\fonts\" >nul 2>&1
if exist "lib\libSkiaSharp.dll" copy "lib\libSkiaSharp.dll" "%DIST_DIR%\lib\" >nul 2>&1
