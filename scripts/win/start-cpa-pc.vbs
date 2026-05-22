Set shell = CreateObject("WScript.Shell")
Set fso = CreateObject("Scripting.FileSystemObject")

appDir = fso.GetParentFolderName(WScript.ScriptFullName)
exePath = fso.BuildPath(appDir, "cpa-pc.exe")

shell.CurrentDirectory = appDir
shell.Run Chr(34) & exePath & Chr(34), 0, False
