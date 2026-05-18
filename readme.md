# ssht

**ssht** or **ssh-tmux**.
As a DevOps engineer, every day i used to
open a *tmux*, create many *panes* and in each of pane i *ssh*
to a host then i set `syncronize-pane on` to use all the
panes at once.

Then I get familiar with [sshs](https://terminaltrove.com/feed/sshs/)
tool. it's just a TUI over
[*ssh_config*](https://man.freebsd.org/cgi/man.cgi?query=ssh_config&sektion=5&manpath=OpenBSD+3.4) file, which let you
search and connect to your ssh host.

I combine this two idea, which the result came out
with `ssht`, a TUI that let you search between your ssh hosts 
and then open all the hosts that you searched
in a tmux session. or you can connect to one host.

### shortcuts
|  Key   |                       Task                       |
|:------:|:------------------------------------------------:|
| CTRL+G |                 Open Guide Menu                  |
| CTRL+A |         Open All Searched Hosts in tmux          |
| CTRL+O | Open All Searched Hosts With Synced Pane In tmux |
| CTRL+N |  Open All Searched Hosts In New Window On tmux   |
| CTRL+C |                       Quit                       |
|  Esc   |                       Back                       |

