trap {
  write-error $_
  exit 1
}

go.exe version

go.exe run github.com/onsi/ginkgo/v2/ginkgo -r -keep-going
if ($LastExitCode -ne 0)
{
    Write-Error $_
    exit 1
}
