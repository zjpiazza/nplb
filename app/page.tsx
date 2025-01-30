"use client";

import { Github, Search, Package, Terminal } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";

export default function Home() {
  return (
    <div className="min-h-screen bg-[#f0f3f5]">
      {/* Header */}
      <header className="bg-[#112e51] text-white py-6 border-b-4 border-[#e31c3d]">
        <div className="container mx-auto px-4">
          <div className="flex items-center justify-between">
            <h1 className="text-2xl font-bold">No Package Left Behind</h1>
            <nav className="space-x-6">
              <a href="#how-it-works" className="hover:text-[#e31c3d] transition">How It Works</a>
              <a href="#search" className="hover:text-[#e31c3d] transition">Find Packages</a>
              <a href="#browse" className="hover:text-[#e31c3d] transition">Browse All</a>
            </nav>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="bg-[#112e51] text-white py-20">
        <div className="container mx-auto px-4 text-center">
          <h2 className="text-4xl font-bold mb-6">Install Any GitHub Project via APT</h2>
          <p className="text-xl mb-8 max-w-2xl mx-auto">
            Found a great GitHub project but no APT repository? We've got you covered. Install any project through your package manager.
          </p>
          <div className="flex justify-center gap-4">
            <Button 
              size="lg"
              className="bg-[#e31c3d] hover:bg-[#cd2026] text-white"
            >
              <Search className="mr-2 h-5 w-5" />
              Find a Package
            </Button>
            <Button 
              size="lg"
              variant="outline"
              className="bg-transparent border-white text-white hover:bg-white hover:text-[#112e51]"
            >
              <Terminal className="mr-2 h-5 w-5" />
              View Instructions
            </Button>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="py-20">
        <div className="container mx-auto px-4">
          <div className="grid md:grid-cols-3 gap-8">
            <Card className="p-6 shadow-lg">
              <h3 className="text-xl font-bold mb-4 text-[#112e51] flex items-center">
                <Search className="mr-2 h-5 w-5" />
                Easy Discovery
              </h3>
              <p className="text-gray-600">
                Search for any GitHub project and install it through APT. No more manual compilation required.
              </p>
            </Card>
            <Card className="p-6 shadow-lg">
              <h3 className="text-xl font-bold mb-4 text-[#112e51] flex items-center">
                <Package className="mr-2 h-5 w-5" />
                Automatic Updates
              </h3>
              <p className="text-gray-600">
                Get updates through your package manager just like any other software on your system.
              </p>
            </Card>
            <Card className="p-6 shadow-lg">
              <h3 className="text-xl font-bold mb-4 text-[#112e51] flex items-center">
                <Terminal className="mr-2 h-5 w-5" />
                Simple Installation
              </h3>
              <p className="text-gray-600">
                Just add our repository and install packages with familiar apt commands.
              </p>
            </Card>
          </div>
        </div>
      </section>

      {/* How It Works */}
      <section id="how-it-works" className="bg-white py-20">
        <div className="container mx-auto px-4">
          <h2 className="text-3xl font-bold text-center mb-12 text-[#112e51]">How It Works</h2>
          <div className="max-w-3xl mx-auto">
            <div className="space-y-8">
              <div className="flex items-start">
                <div className="bg-[#112e51] text-white w-8 h-8 rounded-full flex items-center justify-center font-bold mr-4 flex-shrink-0">1</div>
                <div>
                  <h3 className="text-xl font-bold mb-2 text-[#112e51]">Add Our Repository</h3>
                  <p className="text-gray-600">Add our APT repository to your system with a single command.</p>
                </div>
              </div>
              <div className="flex items-start">
                <div className="bg-[#112e51] text-white w-8 h-8 rounded-full flex items-center justify-center font-bold mr-4 flex-shrink-0">2</div>
                <div>
                  <h3 className="text-xl font-bold mb-2 text-[#112e51]">Find Your Package</h3>
                  <p className="text-gray-600">Search for any GitHub project in our package database.</p>
                </div>
              </div>
              <div className="flex items-start">
                <div className="bg-[#112e51] text-white w-8 h-8 rounded-full flex items-center justify-center font-bold mr-4 flex-shrink-0">3</div>
                <div>
                  <h3 className="text-xl font-bold mb-2 text-[#112e51]">Install & Enjoy</h3>
                  <p className="text-gray-600">Install the package using apt and keep it updated automatically.</p>
                </div>
              </div>
            </div>
          </div>
        </div>
      </section>

      {/* Example Usage */}
      <section className="py-20 bg-[#112e51] text-white">
        <div className="container mx-auto px-4">
          <h2 className="text-3xl font-bold text-center mb-12">Quick Start</h2>
          <div className="max-w-2xl mx-auto bg-[#0a1f35] rounded-lg p-6">
            <p className="text-gray-300 mb-4">Add our repository:</p>
            <pre className="bg-black rounded p-4 mb-6 overflow-x-auto">
              <code>curl -s https://apt.nopackageleftbehind.org/key.gpg | sudo apt-key add -
sudo add-apt-repository "deb https://apt.nopackageleftbehind.org stable main"</code>
            </pre>
            <p className="text-gray-300 mb-4">Install a package:</p>
            <pre className="bg-black rounded p-4 overflow-x-auto">
              <code>sudo apt update
sudo apt install github-awesome-project</code>
            </pre>
          </div>
        </div>
      </section>

      {/* Footer */}
      <footer className="bg-[#112e51] text-white py-12 border-t border-[#1b4971]">
        <div className="container mx-auto px-4">
          <div className="grid md:grid-cols-3 gap-8">
            <div>
              <h3 className="text-xl font-bold mb-4">No Package Left Behind</h3>
              <p className="text-sm text-gray-300">
                Making GitHub projects easily accessible through APT package management.
              </p>
            </div>
            <div>
              <h4 className="font-bold mb-4">Quick Links</h4>
              <ul className="space-y-2">
                <li><a href="#how-it-works" className="text-gray-300 hover:text-white">How It Works</a></li>
                <li><a href="#search" className="text-gray-300 hover:text-white">Find Packages</a></li>
                <li><a href="#browse" className="text-gray-300 hover:text-white">Browse All</a></li>
              </ul>
            </div>
            <div>
              <h4 className="font-bold mb-4">Connect</h4>
              <div className="flex space-x-4">
                <a href="https://github.com" className="text-gray-300 hover:text-white">
                  <Github className="h-6 w-6" />
                </a>
              </div>
            </div>
          </div>
          <div className="border-t border-gray-700 mt-8 pt-8 text-center text-sm text-gray-300">
            © {new Date().getFullYear()} No Package Left Behind. All rights reserved.
          </div>
        </div>
      </footer>
    </div>
  );
}