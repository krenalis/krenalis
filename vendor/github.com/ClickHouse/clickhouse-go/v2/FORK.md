# How to sync with upstream

The idea is not to merge the entire `main` upstream, but only to **merge up to the latest release** of the upstream.

> Note: this requires that the local repository has remotes configured correctly. Check first through the [GitHub guide](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/working-with-forks/configuring-a-remote-repository-for-a-fork).

To do this, from the command line, in our repository, first fetch locally the latest changes of both our fork and the upstream

```
git switch main
git pull
git fetch upstream
```

Then merge the commit of the upstream that has been tagged with the latest tag:

```
git merge <id of commit with last tag on upstream>
```

Finally, simply push the changes to the `main` branch of our fork:

```
git push
```