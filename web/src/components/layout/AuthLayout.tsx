import { Outlet } from "react-router-dom";

export function AuthLayout() {
  return (
    <div className="min-h-screen flex">
      {/* Left side - Branding */}
      <div className="hidden lg:flex lg:w-1/2 bg-primary p-12 flex-col justify-between">
        <div>
          <div className="flex items-center space-x-3">
            <div className="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center">
              <span className="text-white font-bold text-2xl">K</span>
            </div>
            <span className="text-white font-bold text-2xl">K-ERP</span>
          </div>
        </div>

        <div className="space-y-6">
          <h1 className="text-4xl font-bold text-white leading-tight">
            중소기업을 위한
            <br />
            스마트 클라우드 ERP
          </h1>
          <p className="text-white/80 text-lg">
            회계, 세금계산서, 인사급여, 4대보험까지
            <br />
            하나의 플랫폼에서 모두 관리하세요.
          </p>

          <div className="grid grid-cols-2 gap-4 pt-6">
            <div className="bg-white/10 rounded-xl p-4">
              <p className="text-white/60 text-sm">지원 기업 수</p>
              <p className="text-white text-2xl font-bold">1,200+</p>
            </div>
            <div className="bg-white/10 rounded-xl p-4">
              <p className="text-white/60 text-sm">처리 전표 수</p>
              <p className="text-white text-2xl font-bold">50만+</p>
            </div>
            <div className="bg-white/10 rounded-xl p-4">
              <p className="text-white/60 text-sm">세금계산서 발행</p>
              <p className="text-white text-2xl font-bold">100만+</p>
            </div>
            <div className="bg-white/10 rounded-xl p-4">
              <p className="text-white/60 text-sm">고객 만족도</p>
              <p className="text-white text-2xl font-bold">98%</p>
            </div>
          </div>
        </div>

        <div className="text-white/60 text-sm">
          <p>K-ERP SaaS Platform v0.2.0</p>
        </div>
      </div>

      {/* Right side - Form */}
      <div className="flex-1 flex items-center justify-center p-8">
        <div className="w-full max-w-md">
          <Outlet />
        </div>
      </div>
    </div>
  );
}
