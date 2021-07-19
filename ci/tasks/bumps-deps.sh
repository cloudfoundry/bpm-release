set -e #stops the execution if error
cd bpm-release

git config --global user.email "ci@localhost"
git config --global user.name "CI Bot"

runc_version_go_mod=$(grep 'runc' src/bpm/go.mod | sed 's/.* v//')
if ! $(grep -q "$runc_version_go_mod" config/blobs.yml); then
curl -o ./runc_filename.tar.xz -L https://github.com/opencontainers/runc/releases/download/v${runc_version_go_mod}/runc.tar.xz
runc_version_old_versionrsion=$(grep 'runc' config/blobs.yml |  sed 's/.$//')
bosh remove-blob $runc_version_old_versionrsion

bosh add-blob ./runc_filename.tar.xz runc/runc-${runc_version_go_mod}.tar.xz
echo "${BOSH_PRIVATE_CONFIG}" > config/private.yml

bosh upload-blobs

rm runc_filename.tar.xz

go get -u ./...

if [ "$(git status --porcelain)" != "" ]; then
  git add /src/bpm/go.mod /src/bpm/go.sum /config/blobs.yml 
  git commit -m "Update Runc"
fi

fi