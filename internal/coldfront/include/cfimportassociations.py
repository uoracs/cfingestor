#!/usr/bin/env python

import sys
import json

from django.contrib.auth.models import User
from coldfront.core.project.models import Project,ProjectUser,ProjectUserRoleChoice,ProjectUserStatusChoice

DEBUG=True

def dprint(*args):
    if DEBUG:
        print(*args, file=sys.stderr)

ASSOCIATION_DATA_PATH = "/var/run/cfingestor/associations.json"

def get_association_data() -> list[dict]:
    with open(ASSOCIATION_DATA_PATH, "r") as f:
        return json.load(f)

class ProjectUserTracker:
    def __init__(self):
        self.epu = list(ProjectUser.objects.all())

    def tick(self, user, project):
        for i, pu in enumerate(self.epu):
            if pu.user_id == user.id and pu.project_id == project.id:
                self.epu.pop(i)

    def remaining(self) -> list:
        return self.epu


outdata = {"created": [], "removed": []}


# keeps track of the associations we process
# at the end, there may be remaining projects in the tracker
# we need to delete them from coldfront
pu_tracker = ProjectUserTracker()
# dprint("loaded project tracker")

for assoc in get_association_data():
    pname = assoc["fields"]["project"][0]
    powner = assoc["fields"]["project"][1]
    pusername = assoc["fields"]["user"][0]
    prole = assoc["fields"]["role"][0]
    pstatus = assoc["fields"]["status"][0]
    pnotify = assoc["fields"]["enable_notifications"]
    cfuser = User.objects.get(username=pusername)
    cfproj = Project.objects.get(title=pname)
    proleobj = ProjectUserRoleChoice.objects.get(name=prole)
    pstatusobj = ProjectUserStatusChoice.objects.get(name=pstatus)
    # dprint(f"processing association: {pname}:{pusername}")

    try:
        # see if the association already exists
        # if it does, just skip it
        cfprojassoc = ProjectUser.objects.get(
            user_id=cfuser.id,
            project_id=cfproj.id
        )
        # dprint(f"association exists: {pname}:{pusername}")
        pu_tracker.tick(cfuser, cfproj)
    except Exception as e:
        # the association doesnt exist
        # fall through and create it
        # dprint(e)
        # dprint(f"association not found, creating: {pname}:{pusername}")
        cfassoc = ProjectUser.objects.create(
            user=cfuser,
            project=cfproj,
            role=proleobj,
            status=pstatusobj,
            enable_notifications=pnotify
        )
        outdata["created"].append({
            "project": pname,
            "user": pusername,
        })
        pu_tracker.tick(cfuser, cfproj)
        dprint(f"association created: {pname}:{pusername}")

# these associations are present in coldfront not but in AD.
# set them all to "Removed" status
remaining_associations = pu_tracker.remaining()
removed_status = ProjectUserStatusChoice.objects.get(name="Removed")
for ra in remaining_associations:
    if ra.status == removed_status:
        continue
    dprint(f"removing association: {ra.project.title}:{ra.user.username}")
    ra.status = removed_status
    ra.save()
    outdata["removed"].append({
        "project": ra.project.title,
        "user": ra.user.username,
    })

print(json.dumps(outdata))
