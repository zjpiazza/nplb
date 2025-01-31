-- CreateTable
CREATE TABLE "packages" (
    "id" UUID NOT NULL,
    "name" TEXT NOT NULL,
    "version" TEXT NOT NULL,
    "architecture" TEXT NOT NULL,
    "description" TEXT,
    "maintainer" TEXT,
    "depends" TEXT,
    "path" TEXT NOT NULL,
    "storage_url" TEXT NOT NULL,
    "size" INTEGER NOT NULL,
    "md5" TEXT NOT NULL,
    "sha1" TEXT NOT NULL,
    "sha256" TEXT NOT NULL,
    "repositoryId" TEXT NOT NULL,
    "created_at" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "packages_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "repository_metadata" (
    "id" UUID NOT NULL,
    "type" TEXT NOT NULL,
    "content" TEXT NOT NULL,
    "updated_at" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "repository_metadata_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "repositories" (
    "id" TEXT NOT NULL,
    "owner" TEXT NOT NULL,
    "name" TEXT NOT NULL,
    "github_repo" TEXT NOT NULL,
    "github_owner" TEXT NOT NULL,
    "createdAt" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "updatedAt" TIMESTAMP(3) NOT NULL,

    CONSTRAINT "repositories_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE INDEX "packages_repositoryId_idx" ON "packages"("repositoryId");

-- CreateIndex
CREATE INDEX "packages_name_idx" ON "packages"("name");

-- CreateIndex
CREATE INDEX "packages_path_idx" ON "packages"("path");

-- CreateIndex
CREATE UNIQUE INDEX "packages_name_version_architecture_key" ON "packages"("name", "version", "architecture");

-- CreateIndex
CREATE UNIQUE INDEX "repository_metadata_type_key" ON "repository_metadata"("type");

-- CreateIndex
CREATE UNIQUE INDEX "repositories_owner_name_key" ON "repositories"("owner", "name");
