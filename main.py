#!/usr/bin/env python

import logging

logging.basicConfig(level=logging.INFO)

import json
import os

from coldfront.core.allocation.models import (
    Allocation,
    AllocationStatusChoice,
    AllocationUser,
    AllocationUserStatusChoice,
)
from coldfront.core.project.models import (
    Project,
    ProjectStatusChoice,
    ProjectUser,
    ProjectUserRoleChoice,
    ProjectUserStatusChoice,
)
from coldfront.core.resource.models import Resource, ResourceType
from django.contrib.auth.models import User
from django.db.models import Q
from flask import Flask, request

RUN_DIR = "/var/run/cfingestor"
DOMAIN = "uoregon.edu"
RESOURCE_NAME = "Talapas2"
RESOURCE_DESCRIPTION = "University of Oregon HPC Cluster"
ALLOCATION_START_DATE = "2024-01-01"
ALLOCATION_END_DATE = "2024-12-31"


def is_ingest_locked() -> bool:
    return os.path.exists(f"{RUN_DIR}/ingest.lock")


def lock_ingest():
    with open(f"{RUN_DIR}/ingest.lock", "w") as f:
        f.write("")


def unlock_ingest():
    if is_ingest_locked():
        os.remove(f"{RUN_DIR}/ingest.lock")


def exit_error(obj: dict) -> dict:
    unlock_ingest()
    return obj


# unlock ingest if it's locked
if is_ingest_locked():
    unlock_ingest()

# Create /var/run/cfingestor directory if it doesn't exist
if not os.path.exists(RUN_DIR):
    os.makedirs(RUN_DIR)


class ColdfrontModelManager:
    def __init__(self):
        self.coldfront_users = list(User.objects.all())
        self.coldfront_projects = list(Project.objects.all())
        self.coldfront_associations = list(ProjectUser.objects.all())
        self.coldfront_resources = list(Resource.objects.all())
        self.coldfront_allocations = list(Allocation.objects.all())
        self.coldfront_allocation_users = list(AllocationUser.objects.all())

    def refresh_users(self):
        self.coldfront_users = list(User.objects.all())

    def refresh_projects(self):
        self.coldfront_projects = list(Project.objects.all())

    def refresh_associations(self):
        self.coldfront_associations = list(ProjectUser.objects.all())

    def refresh_resources(self):
        self.coldfront_resources = list(Resource.objects.all())

    def refresh_allocations(self):
        self.coldfront_allocations = list(Allocation.objects.all())

    def refresh_allocation_users(self):
        self.coldfront_allocation_users = list(AllocationUser.objects.all())


class UserTracker:
    def __init__(self, users: list):
        self.eu = [u for u in users]

    def tick(self, user):
        for i, u in enumerate(self.eu):
            if u.id == user.id:
                self.eu.pop(i)

    def remaining(self) -> list:
        return self.eu


class ProjectTracker:
    def __init__(self, projects: list):
        self.ep = [p for p in projects]

    def tick(self, project):
        for i, p in enumerate(self.ep):
            if p.id == project.id:
                self.ep.pop(i)

    def remaining(self) -> list:
        return self.ep


class ProjectUserTracker:
    def __init__(self, projectusers: list):
        self.epu = [pu for pu in projectusers]

    def tick(self, user, project):
        for i, pu in enumerate(self.epu):
            if pu.user_id == user.id and pu.project_id == project.id:
                self.epu.pop(i)

    def remaining(self) -> list:
        return self.epu


class ResourceTracker:
    def __init__(self, resources: list):
        self.er = [r for r in resources]

    def tick(self, resource):
        for i, r in enumerate(self.er):
            if r.id == resource.id:
                self.er.pop(i)

    def remaining(self) -> list:
        return self.er


def get_allocation(project: Project, allocations: list) -> Allocation | None:
    for a in allocations:
        if a.project_id == project.id:
            return a


class AllocationTracker:
    def __init__(self, allocations: list):
        self.ea = [a for a in allocations]

    def get(self, project: Project) -> Allocation | None:
        return get_allocation(project, self.ea)

    def tick(self, allocation):
        for i, a in enumerate(self.ea):
            if a.id == allocation.id:
                self.ea.pop(i)

    def remaining(self) -> list:
        return self.ea


class AllocationUserTracker:
    def __init__(self, allocationusers: list):
        self.eau = [au for au in allocationusers]

    def tick(self, user, allocation):
        for i, au in enumerate(self.eau):
            if au.user_id == user.id and au.allocation.id == allocation.id:
                self.eau.pop(i)

    def remaining(self) -> list:
        return self.eau


class ManifestUser:
    def __init__(self, username: str, firstname: str, lastname: str):
        self.username = username
        self.firstname = firstname
        self.lastname = lastname


class ManifestProject:
    def __init__(self, name: str, owner: str, users: list[str], admins: list[str]):
        self.name = name
        self.owner = owner
        self.users = users
        self.admins = admins


class Manifest:
    def __init__(self, users: list[ManifestUser], projects: list[ManifestProject]):
        self.users = users
        self.projects = projects

    def to_json(self) -> str:
        return json.dumps(
            {
                "users": [
                    {
                        "username": u.username,
                        "firstname": u.firstname,
                        "lastname": u.lastname,
                    }
                    for u in self.users
                ],
                "projects": [
                    {
                        "name": p.name,
                        "owner": p.owner,
                        "users": p.users,
                        "admins": p.admins,
                    }
                    for p in self.projects
                ],
            }
        )

    def save_to_file(self, path: str):
        with open(path, "w") as f:
            f.write(self.to_json())

    @staticmethod
    def load_from_file(path: str):
        with open(path, "r") as f:
            return Manifest.from_json(f.read())

    @staticmethod
    def from_json(j: str):
        try:
            data = json.loads(j)
            users = [
                ManifestUser(
                    username=u["username"],
                    firstname=u["firstname"],
                    lastname=u["lastname"],
                )
                for u in data["users"]
            ]
            projects = [
                ManifestProject(
                    name=p["name"],
                    owner=p["owner"],
                    users=p["users"],
                    admins=p["admins"],
                )
                for p in data["projects"]
            ]
            return Manifest(users, projects)
        except Exception as e:
            raise Exception("Error parsing manifest json: " + str(e))


app = Flask(__name__)


@app.route("/manifest", methods=["POST"])
def post_manifest():
    content_hash = request.headers.get("Content-Hash")
    if not content_hash:
        return {"error": "Content-Hash header is required"}, 400
    try:
        # yes i'm serializing and deserializing the json to ensure it's valid
        j = request.json
        if not j:
            return {"error": "JSON body is required"}, 400
        manifest = Manifest.from_json(json.dumps(j))
    except Exception as e:
        return {"error": str(e)}, 400
    return manifest_post_handler(manifest, content_hash)


@app.route("/manifest", methods=["GET"])
def get_manifest():
    return manifest_get_handler()


@app.route("/ingest", methods=["POST"])
def post_ingest():
    return ingest_post_handler()


@app.route("/ingest", methods=["GET"])
def get_ingest():
    return ingest_get_handler()


def get_current_hash() -> str:
    try:
        with open(f"{RUN_DIR}/current_hash", "r") as f:
            return f.read().strip()
    except:
        return ""


def set_current_hash(h: str):
    with open(f"{RUN_DIR}/current_hash", "w") as f:
        f.write(h)


def save_manifest(manifest: Manifest):
    try:
        with open(f"{RUN_DIR}/manifest.json", "w") as f:
            f.write(manifest.to_json())
    except:
        raise


def load_manifest() -> Manifest:
    try:
        with open(f"{RUN_DIR}/manifest.json", "r") as f:
            return Manifest.from_json(f.read())
    except:
        raise


def manifest_post_handler(manifest: Manifest, content_hash: str):
    logging.info("Received POST request on /manifest")

    current_hash = get_current_hash()
    if content_hash == current_hash:
        return exit_error(
            {"status": "Manifest already saved", "hash": content_hash}
        ), 200

    try:
        manifest.save_to_file(f"{RUN_DIR}/manifest.json")
    except:
        return exit_error({"status": "Error saving manifest"}), 500

    try:
        set_current_hash(content_hash)
    except:
        return exit_error({"status": "Error saving hash"}), 500

    logging.info("Manifest saved successfully")
    return exit_error(
        {"status": "Manifest saved successfully", "hash": content_hash}
    ), 201


def manifest_get_handler():
    logging.info("Called GET handler on activedirectory manifest endpoint")

    try:
        manifest = Manifest.load_from_file(f"{RUN_DIR}/manifest.json")
    except:
        return exit_error({"status": "Error loading manifest"}), 500
    return manifest.to_json(), 200


def ingest_get_handler():
    l = is_ingest_locked()
    if l:
        return exit_error({"status": "Ingest is locked"}), 425
    return {"status": "Ingest is not locked"}


def ingest_post_handler():
    logging.info("Received POST request on /ingest")
    l = is_ingest_locked()
    if l:
        return exit_error({"status": "Ingest is locked"}), 425
    lock_ingest()
    logging.info("ingest locked")

    logging.info("reading manifest.json")
    try:
        manifest = Manifest.load_from_file(f"{RUN_DIR}/manifest.json")
    except:
        return exit_error({"status": "Error loading manifest"}), 500
    logging.info("manifest read successfully")

    cfmanager = ColdfrontModelManager()

    logging.info("syncing users")
    user_tracker = UserTracker(cfmanager.coldfront_users)
    for user in manifest.users:
        if user.username not in [u.username for u in cfmanager.coldfront_users]:
            logging.info(f"creating user {user.username}")
            email = f"{user.username}@{DOMAIN}"
            try:
                User.objects.create(
                    email=email,
                    username=user.username,
                    first_name=user.firstname,
                    last_name=user.lastname,
                    is_active=True,
                    is_staff=False,
                    is_superuser=False,
                )
                logging.info(f"created user {user.username}")
            except Exception as e:
                logging.error(f"error creating user {user.username}: {e}")
                return exit_error(
                    {"status": f"Error creating user {user.username}: {e}"}
                ), 500
        cfuser = User.objects.get(username=user.username)
        if not cfuser.is_active:
            logging.info(f"activating user {user.username}")
            cfuser.is_active = True
            cfuser.save()
            logging.info(f"activated user {user.username}")
        user_tracker.tick(cfuser)
    for user in user_tracker.remaining():
        if user.username == "admin":
            # skip django admin
            continue
        if not user.is_active:
            # skip deactivated users
            continue
        logging.info(f"deactivating user {user.username}")
        try:
            user.is_active = False
            user.save()
            logging.info(f"deactivated user {user.username}")
        except Exception as e:
            logging.error(f"error deactivating user {user.username}: {e}")
            return exit_error(
                {"status": f"Error deactivating user {user.username}: {e}"}
            ), 500
    cfmanager.refresh_users()
    logging.info("users synced successfully")

    logging.info("syncing projects")
    project_tracker = ProjectTracker(cfmanager.coldfront_projects)
    project_active_status = ProjectStatusChoice.objects.get(name="Active")
    project_archived_status = ProjectStatusChoice.objects.get(name="Archived")
    for project in manifest.projects:
        if project.name not in [p.title for p in cfmanager.coldfront_projects]:
            logging.info(f"creating project {project.name}")
            try:
                cfpi = User.objects.get(username=project.owner)
                Project.objects.create(
                    title=project.name,
                    pi=cfpi,
                    description="enter description",
                    status=project_active_status,
                    requires_review=False,
                    force_review=False,
                )
                logging.info(f"created project {project.name}")
            except Exception as e:
                logging.error(f"error creating project {project.name}: {e}")
                return exit_error(
                    {"status": f"Error creating project {project.name}: {e}"}
                ), 500
        cfproject = Project.objects.get(title=project.name)
        cfproject.requires_review = True
        cfproject.status = project_active_status
        cfproject.description = "enter description"
        cfproject.save()
        project_tracker.tick(cfproject)
    for project in project_tracker.remaining():
        try:
            logging.info(f"archiving project {project.name}")
            p = Project.objects.get(title=project.title)
            p.status = project_archived_status
            p.save()
            logging.info(f"archiving project {project.name}")
        except Exception as e:
            logging.error(f"error deactivating project {project.name}: {e}")
            return exit_error(
                {"status": f"Error deactivating project {project.name}: {e}"}
            ), 500
    cfmanager.refresh_projects()
    logging.info("projects synced successfully")

    logging.info("syncing associations")
    cf_status_active = ProjectUserStatusChoice.objects.get(name="Active")
    cf_status_inactive = ProjectUserStatusChoice.objects.get(name="Removed")
    association_tracker = ProjectUserTracker(cfmanager.coldfront_associations)
    for manifest_project in manifest.projects:
        logging.info(f"processing project: {manifest_project.name}")
        for username in manifest_project.users:
            logging.info(f"processing user: {username}")
            if manifest_project.owner == username:
                logging.info(f"skipping pi: {username}")
                # skip PIs
                continue
            # coldfront objects
            cf_user = User.objects.get(username=username)
            cf_project = Project.objects.get(title=manifest_project.name)
            cf_role = ProjectUserRoleChoice.objects.get(name="User")
            if username in manifest_project.admins:
                cf_role = ProjectUserRoleChoice.objects.get(name="Manager")
            # find a projectuser with the username and project name
            try:
                assoc = ProjectUser.objects.get(
                    project=cf_project,
                    user=cf_user,
                )
            except:
                logging.info(
                    f"creating association {cf_user.username} -> {cf_project.title}"
                )
                assoc = ProjectUser.objects.create(
                    project=cf_project,
                    user=cf_user,
                    status=cf_status_active,
                    role=cf_role,
                )
                logging.info(
                    f"created association {cf_user.username} -> {cf_project.title}"
                )
            if assoc.role != cf_role:
                logging.info(
                    f"updating association role {cf_user.username} -> {cf_project.title}"
                )
                assoc.role = cf_role
                assoc.save()
                logging.info(
                    f"updated association role {cf_user.username} -> {cf_project.title}"
                )
            if assoc.status != cf_status_active:
                logging.info(
                    f"activating association {cf_user.username} -> {cf_project.title}"
                )
                assoc.status = cf_status_active
                assoc.save()
                logging.info(
                    f"activated association {cf_user.username} -> {cf_project.title}"
                )
            association_tracker.tick(cf_user, cf_project)

    for association in association_tracker.remaining():
        try:
            f"deactivating association {association.user.username} -> {association.project.title}"
            association.status = cf_status_inactive
            association.save()
            f"deactivated association {association.user.username} -> {association.project.title}"
        except Exception as e:
            return {
                "status": f"Error deactivating association {association.user.username} -> {association.project.title}: {e}"
            }, 500
    cfmanager.refresh_associations()
    logging.info("associations synced successfully")

    logging.info("syncing resources")
    try:
        Resource.objects.get(name=RESOURCE_NAME)
    except:
        logging.info(f"creating resource {RESOURCE_NAME}")
        cf_resource_type = ResourceType.objects.get(name="Cluster")
        Resource.objects.create(
            name=RESOURCE_NAME,
            description=RESOURCE_DESCRIPTION,
            is_allocatable=True,
            is_available=True,
            is_public=True,
            requires_payment=False,
            resource_type=[cf_resource_type],
        )
        logging.info(f"created resource {RESOURCE_NAME}")
    logging.info("resources synced successfully")

    logging.info("syncing allocations")
    allocation_tracker = AllocationTracker(cfmanager.coldfront_allocations)
    cluster_resource = Resource.objects.get(name=RESOURCE_NAME)
    cf_alloc_status_active = AllocationStatusChoice.objects.get(name="Active")
    cf_alloc_status_expired = AllocationStatusChoice.objects.get(name="Expired")
    for project in manifest.projects:
        cf_project = Project.objects.get(title=project.name)
        cf_allocation = allocation_tracker.get(cf_project)
        if not cf_allocation:
            try:
                logging.info(f"creating allocation {project.name}")
                cf_allocation = Allocation.objects.create(
                    project=cf_project,
                    start_date=ALLOCATION_START_DATE,
                    end_date=ALLOCATION_END_DATE,
                    status=cf_alloc_status_active,
                )
                cf_allocation.resources.set([cluster_resource])
                cf_allocation.save()
                logging.info(f"created allocation {project.name}")
            except Exception as e:
                logging.error(f"error creating allocation {project.name}: {e}")
                return exit_error(
                    {"status": f"Error creating allocation {project.name}: {e}"}
                ), 500
        allocation_tracker.tick(cf_allocation)
    for remaining_allocation in allocation_tracker.remaining():
        if remaining_allocation.status == cf_alloc_status_expired:
            continue
        try:
            logging.info(
                f"deactivating allocation {remaining_allocation.project.title}"
            )
            remaining_allocation.status = cf_alloc_status_expired
            remaining_allocation.save()
            logging.info(f"deactivated allocation {remaining_allocation.project.title}")
        except Exception as e:
            logging.error(
                f"error deactivating allocation {remaining_allocation.project.title}: {e}"
            )
            return {
                "status": f"Error deactivating allocation {remaining_allocation.project.title}: {e}"
            }, 500
    cfmanager.refresh_allocations()
    logging.info("allocations synced successfully")

    logging.info("syncing allocation users")
    allocation_user_tracker = AllocationUserTracker(
        cfmanager.coldfront_allocation_users
    )
    cf_alloc_user_status_active = AllocationUserStatusChoice.objects.get(name="Active")
    cf_alloc_user_status_removed = AllocationUserStatusChoice.objects.get(
        name="Removed"
    )
    for project in manifest.projects:
        cf_project = Project.objects.get(title=project.name)
        for username in project.users:
            if project.owner == username:
                # skip PIs
                continue
            cf_user = User.objects.get(username=username)
            cf_allocation = get_allocation(cf_project, cfmanager.coldfront_allocations)
            try:
                allocation_user = AllocationUser.objects.get(
                    allocation=cf_allocation,
                    user=cf_user,
                )
            except:
                logging.info(
                    f"creating allocation user {cf_user.username} -> {cf_project.title}"
                )
                allocation_user = AllocationUser.objects.create(
                    allocation=cf_allocation,
                    user=cf_user,
                    status=cf_alloc_user_status_active,
                )
                logging.info(
                    f"created allocation user {cf_user.username} -> {cf_project.title}"
                )
            allocation_user_tracker.tick(cf_user, cf_allocation)
    for allocation_user in allocation_user_tracker.remaining():
        try:
            f"removing allocation user {allocation_user.user.username} -> {allocation_user.allocation.project.title}"
            allocation_user.status = cf_alloc_user_status_removed
            allocation_user.save()
        except Exception as e:
            return {
                "status": f"Error deactivating allocation user {allocation_user.user.username} -> {allocation_user.allocation.project.title}: {e}"
            }, 500
    cfmanager.refresh_allocation_users()
    logging.info("allocation users synced successfully")

    unlock_ingest()
    logging.info("ingest unlocked")
    return {"status": "Ingest completed successfully"}, 200


app.run(host="0.0.0.0", port=8090)
