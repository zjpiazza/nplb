"use client";

import { Github, Search, Package, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import CreateRepoForm from '@/components/CreateRepoForm'

export default function Home() {
  return (
    <main className="min-h-screen p-8">
      <h1 className="text-3xl font-bold mb-8">APT Repository Generator</h1>
      <CreateRepoForm />
    </main>
  );
}